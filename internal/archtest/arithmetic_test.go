package archtest

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
)

// This file implements the three arithmetic rules DESIGN.md §5 places in
// archtest alongside the import backstop: the product-rounding rule, the closed
// math allowlist, and the float-to-integer conversion gate. depguard cannot
// express any of them, because all three need types rather than import paths.

const (
	ruleProduct  = "product"
	ruleMath     = "math"
	ruleFloatInt = "floatint"

	domainPackagePath = module + "/internal/domain"
	domainPathPrefix  = "internal/domain/"

	// The one audited float-to-integer conversion, exempted by name.
	roundPopulationFile = "internal/domain/quantities.go"
	roundPopulationFunc = "RoundPopulation"
)

// productionMathAllowlist is §5's closed list: bit manipulation or exact
// comparisons whose results are fixed for the values their call sites admit.
// Adding an entry requires the admission proof §5 states, not a precedent.
var productionMathAllowlist = map[string]bool{
	"Abs":             true,
	"Copysign":        true,
	"Float64bits":     true,
	"Float64frombits": true,
	"IsInf":           true,
	"IsNaN":           true,
	"Max":             true,
	"Min":             true,
	"Signbit":         true,
}

// testOnlyMathAllowlist may be used only to check the generated tables against
// the functions that produced them, and never from a production path.
var testOnlyMathAllowlist = map[string]bool{"Sin": true, "Cos": true}

type arithmeticViolation struct {
	Rule string
	File string
	Line int
	Text string
}

func (v arithmeticViolation) String() string {
	return v.File + ":" + itoa(v.Line) + ": [" + v.Rule + "] " + v.Text
}

// arithmeticReport carries the exemption counters as well as the violations so
// that a fixture can assert an expression was accepted *for the stated reason*
// rather than accidentally, which is the whole point of the constant exemption.
type arithmeticReport struct {
	Violations     []arithmeticViolation
	ConstantExempt int
	ChainExempt    int
}

func (r arithmeticReport) rules() []string {
	seen := make([]string, 0, len(r.Violations))
	for _, violation := range r.Violations {
		seen = append(seen, violation.Rule)
	}
	sort.Strings(seen)
	return seen
}

func (r arithmeticReport) String() string {
	lines := make([]string, 0, len(r.Violations))
	for _, violation := range r.Violations {
		lines = append(lines, violation.String())
	}
	return strings.Join(lines, "\n")
}

type analyzedFile struct {
	path string
	file *ast.File
}

func analyzeArithmetic(fset *token.FileSet, info *types.Info, files []analyzedFile) arithmeticReport {
	report := arithmeticReport{}
	for _, analyzed := range files {
		analyzer := &arithmeticAnalyzer{
			fset:   fset,
			info:   info,
			path:   analyzed.path,
			isTest: strings.HasSuffix(analyzed.path, "_test.go"),
			report: &report,
		}
		ast.Inspect(analyzed.file, analyzer.visit)
	}
	sort.Slice(report.Violations, func(i, j int) bool {
		left, right := report.Violations[i], report.Violations[j]
		if left.File != right.File {
			return left.File < right.File
		}
		if left.Line != right.Line {
			return left.Line < right.Line
		}
		return left.Rule < right.Rule
	})
	return report
}

type arithmeticAnalyzer struct {
	fset   *token.FileSet
	info   *types.Info
	path   string
	isTest bool
	report *arithmeticReport
	stack  []ast.Node
}

func (a *arithmeticAnalyzer) visit(node ast.Node) bool {
	if node == nil {
		a.stack = a.stack[:len(a.stack)-1]
		return false
	}
	a.stack = append(a.stack, node)
	switch typed := node.(type) {
	case *ast.BinaryExpr:
		if typed.Op == token.MUL {
			a.checkProduct(typed)
		}
	case *ast.AssignStmt:
		if typed.Tok == token.MUL_ASSIGN {
			a.checkCompoundProduct(typed)
		}
	case *ast.CallExpr:
		a.checkFloatToInteger(typed)
	case *ast.SelectorExpr:
		a.checkMathAllowlist(typed)
	}
	return true
}

// parent returns the closest enclosing node, seeing through parentheses so that
// float64((a*b)) is the same expression as float64(a*b) to this rule.
func (a *arithmeticAnalyzer) parent() ast.Node {
	for i := len(a.stack) - 2; i >= 0; i-- {
		if _, parenthesized := a.stack[i].(*ast.ParenExpr); parenthesized {
			continue
		}
		return a.stack[i]
	}
	return nil
}

func (a *arithmeticAnalyzer) add(rule string, node ast.Node, text string) {
	a.report.Violations = append(a.report.Violations, arithmeticViolation{
		Rule: rule,
		File: a.path,
		Line: a.fset.Position(node.Pos()).Line,
		Text: text,
	})
}

// checkProduct is the product-rounding rule. Go may fuse a*b + c into a single
// operation, and may do so across statements, so the rule is syntactic: a
// floating product is rejected unless its immediate parent is an explicit
// conversion to the product's own type. Two exemptions apply, each for a reason
// that is about where the hazard is rather than about convenience.
func (a *arithmeticAnalyzer) checkProduct(expr *ast.BinaryExpr) {
	if !a.inDomain() {
		return
	}
	recorded, ok := a.info.Types[expr]
	if !ok {
		return
	}
	if parameter, generic := recorded.Type.(*types.TypeParam); generic {
		if typeSetAdmitsFloating(parameter) {
			a.add(ruleProduct, expr, "generic floating multiplication is outside the domain arithmetic contract")
		}
		return
	}
	if !isFloatingType(recorded.Type) {
		return
	}
	// First exemption: the type-checker already folded this product exactly, so
	// no runtime multiplication exists for a fused multiply-add to absorb.
	if recorded.Value != nil {
		a.report.ConstantExempt++
		return
	}
	parent := a.parent()
	// Second exemption: in a*b*c only the outermost product can reach a sum, and
	// no fused multiply-multiply operation exists on the supported targets.
	if outer, chained := parent.(*ast.BinaryExpr); chained && outer.Op == token.MUL {
		a.report.ChainExempt++
		return
	}
	if call, converted := parent.(*ast.CallExpr); converted &&
		a.isConversionTo(call, recorded.Type) &&
		len(call.Args) == 1 && unparenthesize(call.Args[0]) == ast.Expr(expr) {
		return
	}
	a.add(ruleProduct, expr, "floating product must end in an explicit conversion to "+recorded.Type.String())
}

// checkCompoundProduct covers the form the expression walk cannot see. x *= y
// is not a BinaryExpr and contains no multiplication expression, yet
// x *= y; r := x + c is exactly the cross-statement shape the rule exists to
// catch. There is no conversion-wrapped spelling of a compound assignment, so
// every floating one is rejected in favour of the explicit x = T(x * y) form.
func (a *arithmeticAnalyzer) checkCompoundProduct(stmt *ast.AssignStmt) {
	if !a.inDomain() {
		return
	}
	for _, target := range stmt.Lhs {
		targetType := a.info.TypeOf(target)
		if targetType == nil {
			continue
		}
		if parameter, generic := targetType.(*types.TypeParam); generic {
			if typeSetAdmitsFloating(parameter) {
				a.add(ruleProduct, stmt, "generic floating multiplication is outside the domain arithmetic contract")
			}
			continue
		}
		if isFloatingType(targetType) {
			a.add(ruleProduct, stmt, "floating *= has no rounding barrier; write the explicit conversion form")
		}
	}
}

// checkMathAllowlist resolves the selector through types rather than matching
// the identifier "math", so importing the package under an alias does not
// bypass the closed list.
func (a *arithmeticAnalyzer) checkMathAllowlist(selector *ast.SelectorExpr) {
	if !a.inDomain() {
		return
	}
	function, isFunction := a.info.Uses[selector.Sel].(*types.Func)
	if !isFunction || function.Pkg() == nil || function.Pkg().Path() != "math" {
		return
	}
	name := function.Name()
	if productionMathAllowlist[name] {
		return
	}
	if a.isTest && testOnlyMathAllowlist[name] {
		return
	}
	a.add(ruleMath, selector, "math."+name+" is not on the domain allowlist")
}

// checkFloatToInteger closes the cross-target divergence class the other two
// rules leave open: Go makes an out-of-range float-to-integer conversion
// implementation-dependent, so the domain routes every one through the single
// range-checked helper.
func (a *arithmeticAnalyzer) checkFloatToInteger(call *ast.CallExpr) {
	if !a.inDomain() || len(call.Args) != 1 {
		return
	}
	conversion, ok := a.info.Types[call.Fun]
	if !ok || !conversion.IsType() || !isIntegerType(conversion.Type) {
		return
	}
	operand := a.info.TypeOf(unparenthesize(call.Args[0]))
	if operand == nil || !isFloatingType(operand) {
		return
	}
	// A constant conversion is folded exactly by the compiler, so Go's
	// implementation-dependent out-of-range case cannot arise.
	if recorded, known := a.info.Types[call]; known && recorded.Value != nil {
		return
	}
	if declaration := a.enclosingFunction(); a.path == roundPopulationFile &&
		declaration != nil && declaration.Name.Name == roundPopulationFunc &&
		auditedRoundPopulation(a.info, declaration) {
		return
	}
	a.add(ruleFloatInt, call, "float-to-integer conversion must go through RoundPopulation")
}

// enclosingFunction derives scope from the active AST stack instead of keeping
// a sticky last-seen function name. Package declarations after RoundPopulation
// must not inherit its exemption.
func (a *arithmeticAnalyzer) enclosingFunction() *ast.FuncDecl {
	for index := len(a.stack) - 1; index >= 0; index-- {
		if declaration, ok := a.stack[index].(*ast.FuncDecl); ok {
			return declaration
		}
	}
	return nil
}

// auditedRoundPopulation makes the sole exemption depend on the safety proof,
// not just a convenient file and function spelling. The helper must reject both
// non-finite classes and both sides of Population's closed numeric range.
func auditedRoundPopulation(info *types.Info, declaration *ast.FuncDecl) bool {
	if declaration == nil || declaration.Type.Params == nil || declaration.Type.Params.NumFields() > 2 {
		return false
	}
	// The audited value is the sole float64 parameter, identified by type rather
	// than by position. A second parameter is permitted so the conversion can
	// draw from the aggregate's WorldRNG, but it earns no exemption of its own:
	// every proof below still binds to the value being converted, and a helper
	// with no float64 parameter or with two of them is refused outright.
	var parameter types.Object
	for _, field := range declaration.Type.Params.List {
		for _, name := range field.Names {
			object := info.Defs[name]
			if object == nil {
				return false
			}
			basic, isBasic := object.Type().(*types.Basic)
			if !isBasic || basic.Kind() != types.Float64 {
				continue
			}
			if parameter != nil {
				return false
			}
			parameter = object
		}
	}
	if parameter == nil {
		return false
	}
	var hasNaN, hasInf, hasLowerBound, hasUpperBound bool
	ast.Inspect(declaration.Body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.CallExpr:
			if selector, ok := typed.Fun.(*ast.SelectorExpr); ok {
				if function, ok := info.Uses[selector.Sel].(*types.Func); ok &&
					function.Pkg() != nil && function.Pkg().Path() == "math" {
					switch function.Name() {
					case "IsNaN":
						hasNaN = callUsesObject(info, typed, parameter)
					case "IsInf":
						hasInf = callUsesObject(info, typed, parameter)
					}
				}
			}
		case *ast.BinaryExpr:
			if typed.Op == token.LSS && exprUsesObject(info, typed.X, parameter) && isZeroConstant(info, typed.Y) {
				hasLowerBound = true
			}
			if typed.Op == token.GTR && exprUsesObject(info, typed.X, parameter) && exprNames(typed.Y, "MaxPopulation") {
				hasUpperBound = true
			}
		}
		return true
	})
	return hasNaN && hasInf && hasLowerBound && hasUpperBound
}

func callUsesObject(info *types.Info, call *ast.CallExpr, object types.Object) bool {
	for _, argument := range call.Args {
		if exprUsesObject(info, argument, object) {
			return true
		}
	}
	return false
}

func exprUsesObject(info *types.Info, expression ast.Expr, object types.Object) bool {
	found := false
	ast.Inspect(expression, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if ok && info.Uses[identifier] == object {
			found = true
			return false
		}
		return !found
	})
	return found
}

func isZeroConstant(info *types.Info, expression ast.Expr) bool {
	recorded, ok := info.Types[unparenthesize(expression)]
	return ok && recorded.Value != nil && recorded.Value.ExactString() == "0"
}

func exprNames(expression ast.Expr, name string) bool {
	found := false
	ast.Inspect(expression, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if ok && identifier.Name == name {
			found = true
			return false
		}
		return !found
	})
	return found
}

func (a *arithmeticAnalyzer) inDomain() bool {
	return strings.HasPrefix(a.path, domainPathPrefix)
}

func (a *arithmeticAnalyzer) isConversionTo(call *ast.CallExpr, target types.Type) bool {
	recorded, ok := a.info.Types[call.Fun]
	return ok && recorded.IsType() && types.Identical(recorded.Type, target)
}

func unparenthesize(expr ast.Expr) ast.Expr {
	for {
		parenthesized, ok := expr.(*ast.ParenExpr)
		if !ok {
			return expr
		}
		expr = parenthesized.X
	}
}

func isFloatingType(t types.Type) bool {
	basic, ok := t.Underlying().(*types.Basic)
	return ok && basic.Info()&types.IsFloat != 0
}

func isIntegerType(t types.Type) bool {
	basic, ok := t.Underlying().(*types.Basic)
	return ok && basic.Info()&types.IsInteger != 0
}

func typeSetAdmitsFloating(parameter *types.TypeParam) bool {
	constraint, ok := parameter.Constraint().Underlying().(*types.Interface)
	if !ok {
		return isFloatingType(parameter.Constraint())
	}
	return interfaceAdmitsFloating(constraint)
}

func interfaceAdmitsFloating(iface *types.Interface) bool {
	for i := range iface.NumEmbeddeds() {
		switch embedded := iface.EmbeddedType(i).(type) {
		case *types.Union:
			for term := range embedded.Len() {
				if isFloatingType(embedded.Term(term).Type()) {
					return true
				}
			}
		case *types.Interface:
			if interfaceAdmitsFloating(embedded) {
				return true
			}
		default:
			if isFloatingType(embedded) {
				return true
			}
		}
	}
	return false
}

func newTypeInfo() *types.Info {
	return &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
}

// The source importer needs no build artifacts, so this gate stays runnable
// from a clean checkout with nothing installed. It is shared because
// re-importing the standard library per fixture dominates the runtime.
var (
	importerOnce   sync.Once
	sharedImporter types.Importer
)

func sourceImporter() types.Importer {
	importerOnce.Do(func() {
		sharedImporter = importer.ForCompiler(token.NewFileSet(), "source", nil)
	})
	return sharedImporter
}

func analyzeFixture(t *testing.T, path, source string) arithmeticReport {
	t.Helper()
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, path, source, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse fixture %s: %v", path, err)
	}
	info := newTypeInfo()
	config := types.Config{Importer: sourceImporter()}
	if _, err := config.Check(domainPackagePath, fset, []*ast.File{parsed}, info); err != nil {
		t.Fatalf("fixture %s must type-check before it can be analyzed: %v", path, err)
	}
	return analyzeArithmetic(fset, info, []analyzedFile{{path: path, file: parsed}})
}

// analyzeDomainPackage type-checks the real internal/domain package. §5 notes
// this stays cheap because the domain imports only the standard library and
// the dependency-free local gameapi policy contract.
func analyzeDomainPackage(t *testing.T) arithmeticReport {
	t.Helper()
	root := repositoryRoot(t)
	directory := filepath.Join(root, "internal", "domain")
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var (
		parsed  []*ast.File
		visited []analyzedFile
	)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(directory, entry.Name()), nil, parser.AllErrors)
		if err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		parsed = append(parsed, file)
		visited = append(visited, analyzedFile{path: domainPathPrefix + entry.Name(), file: file})
	}
	info := newTypeInfo()
	config := types.Config{Importer: sourceImporter()}
	if _, err := config.Check(domainPackagePath, fset, parsed, info); err != nil {
		t.Fatalf("type-check internal/domain: %v", err)
	}
	return analyzeArithmetic(fset, info, visited)
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := make([]byte, 0, 8)
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}

// TestDomainArithmeticRules is the hard gate: the real package, all three rules.
func TestDomainArithmeticRules(t *testing.T) {
	report := analyzeDomainPackage(t)
	if len(report.Violations) != 0 {
		t.Fatalf("internal/domain arithmetic violations (%d):\n%s", len(report.Violations), report)
	}
}
