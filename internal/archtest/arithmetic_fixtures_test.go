package archtest

import (
	"slices"
	"testing"
)

// The accept/reject fixtures DESIGN.md §12 step 1 enumerates for the three
// arithmetic rules. Each case states the exact source form the rule must judge,
// so the gate's boundary is pinned by example rather than by its implementation.

type arithmeticFixture struct {
	name string
	// path defaults to a production domain file; set it to place the fixture in
	// a _test.go file, in quantities.go, or outside the domain entirely.
	path   string
	source string
	// wantRules lists one rule name per expected violation, sorted.
	wantRules []string
	// wantConstantExempt and wantChainExempt assert an accepted product was
	// accepted for the stated reason rather than by accident.
	wantConstantExempt int
	wantChainExempt    int
}

var arithmeticFixtures = []arithmeticFixture{
	// ---- product rule: rejection, expression cases ----
	{
		name: "reject/product_then_sum",
		source: `package domain
func f(a, b, c float64) float64 { return a*b + c }`,
		wantRules: []string{ruleProduct},
	},
	{
		name: "reject/sum_then_product",
		source: `package domain
func f(a, b, c float64) float64 { return a + b*c }`,
		wantRules: []string{ruleProduct},
	},
	{
		name: "reject/two_products_in_one_sum",
		source: `package domain
func f(a, b, c, d float64) float64 { return a*b + c*d }`,
		wantRules: []string{ruleProduct, ruleProduct},
	},
	{
		name: "reject/product_into_add_assign",
		source: `package domain
func f(a, b float64) float64 {
	x := 0.0
	x += a * b
	return x
}`,
		wantRules: []string{ruleProduct},
	},
	{
		name: "reject/product_bound_then_summed",
		source: `package domain
func f(a, b, c float64) float64 {
	p := a * b
	r := p + c
	return r
}`,
		wantRules: []string{ruleProduct},
	},
	{
		name: "reject/product_through_pointer_then_summed",
		source: `package domain
func f(p *float64, a, b, c float64) float64 {
	*p = a * b
	r := *p + c
	return r
}`,
		wantRules: []string{ruleProduct},
	},
	{
		name: "reject/conversion_wraps_the_sum_not_the_product",
		source: `package domain
func f(a, b, c float64) float64 { return float64(a*b + c) }`,
		wantRules: []string{ruleProduct},
	},
	{
		name: "reject/raw_chain_end_after_inner_conversion",
		source: `package domain
func f(a, b, c float64) float64 { return float64(a*b) * c }`,
		wantRules: []string{ruleProduct},
	},

	// ---- product rule: rejection, compound-assignment cases ----
	{
		name: "reject/compound_float64",
		source: `package domain
func f(x, y float64) float64 {
	x *= y
	return x
}`,
		wantRules: []string{ruleProduct},
	},
	{
		name: "reject/compound_then_summed_across_statements",
		source: `package domain
func f(x, y, c float64) float64 {
	x *= y
	r := x + c
	return r
}`,
		wantRules: []string{ruleProduct},
	},
	{
		name: "reject/compound_float32",
		source: `package domain
func f(x, y float32) float32 {
	x *= y
	return x
}`,
		wantRules: []string{ruleProduct},
	},
	{
		name: "reject/compound_named_floating",
		source: `package domain
type ratio float64
func f(x, y ratio) ratio {
	x *= y
	return x
}`,
		wantRules: []string{ruleProduct},
	},
	{
		name: "reject/compound_indexed_left_side",
		source: `package domain
func f(s []float64, y float64) {
	s[0] *= y
}`,
		wantRules: []string{ruleProduct},
	},
	{
		name: "reject/compound_map_indexed_left_side",
		source: `package domain
func f(m map[string]float64, k string, y float64) {
	m[k] *= y
}`,
		wantRules: []string{ruleProduct},
	},

	// ---- product rule: generic arithmetic is rejected outright ----
	{
		name: "reject/type_parameter_admitting_floating",
		source: `package domain
func f[T ~float64](a, b T) T { return a * b }`,
		wantRules: []string{ruleProduct},
	},
	{
		name: "reject/type_parameter_admitting_floating_even_when_converted",
		source: `package domain
func f[T ~float64](a, b T) T { return T(a * b) }`,
		wantRules: []string{ruleProduct},
	},
	{
		name: "reject/type_parameter_compound_admitting_floating",
		source: `package domain
func f[T float32 | float64](x, y T) T {
	x *= y
	return x
}`,
		wantRules: []string{ruleProduct},
	},
	{
		name: "accept/type_parameter_admitting_only_integers",
		source: `package domain
func f[T ~int | ~int64](a, b T) T { return a * b }`,
		wantRules: []string{},
	},

	// ---- math allowlist ----
	{
		name: "reject/aliased_math_fma",
		source: `package domain
import m "math"
func f(a, b, c float64) float64 { return m.FMA(a, b, c) }`,
		wantRules: []string{ruleMath},
	},
	{
		name: "reject/production_math_sin",
		source: `package domain
import "math"
func f(x float64) float64 { return math.Sin(x) }`,
		wantRules: []string{ruleMath},
	},
	{
		name: "reject/production_math_sqrt_is_not_pre_approved",
		source: `package domain
import "math"
func f(x float64) float64 { return math.Sqrt(x) }`,
		wantRules: []string{ruleMath},
	},
	{
		name: "accept/every_approved_math_call",
		source: `package domain
import "math"
func f(x, y float64) float64 {
	if math.IsNaN(x) || math.IsInf(y, 0) || math.Signbit(x) {
		return math.Abs(x)
	}
	bits := math.Float64bits(x)
	restored := math.Float64frombits(bits)
	return math.Max(math.Min(restored, y), math.Copysign(x, y))
}`,
		wantRules: []string{},
	},
	{
		name: "accept/math_package_constants_are_not_calls",
		source: `package domain
import "math"
func f() float64 { return math.Pi + math.Sqrt2 }`,
		wantRules: []string{},
	},
	{
		name: "accept/test_only_sin_cos_table_check",
		path: "internal/domain/temperature_test.go",
		source: `package domain
import "math"
func check(angle float64) float64 { return math.Sin(angle) + math.Cos(angle) }`,
		wantRules: []string{},
	},
	{
		name: "reject/test_files_get_no_wider_math_licence",
		path: "internal/domain/temperature_test.go",
		source: `package domain
import "math"
func check(x float64) float64 { return math.Pow(x, 2) }`,
		wantRules: []string{ruleMath},
	},

	// ---- product rule: acceptance ----
	{
		name: "accept/converted_product_then_sum",
		source: `package domain
func f(a, b, c float64) float64 { return float64(a*b) + c }`,
		wantRules: []string{},
	},
	{
		name: "accept/converted_product_bound_then_summed",
		source: `package domain
func f(a, b, c float64) float64 {
	p := float64(a * b)
	r := p + c
	return r
}`,
		wantRules: []string{},
	},
	{
		name: "accept/type_appropriate_float32_rewrite",
		source: `package domain
func f(a, b, c float32) float32 { return float32(a*b) + c }`,
		wantRules: []string{},
	},
	{
		name: "accept/type_appropriate_named_floating_rewrite",
		source: `package domain
type ratio float64
func f(a, b, c ratio) ratio { return ratio(a*b) + c }`,
		wantRules: []string{},
	},
	{
		name: "reject/conversion_to_a_different_floating_type",
		source: `package domain
type ratio float64
func f(a, b ratio) float64 { return float64(a * b) }`,
		wantRules: []string{ruleProduct},
	},
	{
		name: "accept/integer_product",
		source: `package domain
func f(a, b, c int) int { return a*b + c }`,
		wantRules: []string{},
	},
	{
		name: "accept/integer_compound_product",
		source: `package domain
func f(x, y int) int {
	x *= y
	return x
}`,
		wantRules: []string{},
	},

	// ---- product rule: the multiplication exemption's exact boundary ----
	{
		name: "accept/chained_product_needs_one_conversion",
		source: `package domain
func f(a, b, c, d float64) float64 { return float64(a*b*c) + d }`,
		wantRules:       []string{},
		wantChainExempt: 1,
	},
	{
		name: "accept/chain_conversion_without_a_sum",
		source: `package domain
func f(a, b, c float64) float64 { return float64(a * b * c) }`,
		wantRules:       []string{},
		wantChainExempt: 1,
	},
	{
		name: "accept/redundant_inner_conversion_remains_legal",
		source: `package domain
func f(a, b, c float64) float64 { return float64(float64(a*b) * c) }`,
		wantRules: []string{},
	},

	// ---- product rule: the constant exemption ----
	{
		name: "accept/bare_untyped_constant_product",
		source: `package domain
const X = 0.5 * 2.0`,
		wantRules:          []string{},
		wantConstantExempt: 1,
	},
	{
		name: "accept/typed_float64_constant_product",
		source: `package domain
const X float64 = 0.5 * 2.0`,
		wantRules:          []string{},
		wantConstantExempt: 1,
	},
	{
		name: "accept/typed_float32_constant_product",
		source: `package domain
const X float32 = 0.5 * 2.0`,
		wantRules:          []string{},
		wantConstantExempt: 1,
	},
	{
		name: "accept/constant_product_in_a_configuration_table",
		source: `package domain
type entry struct {
	Rate  float64
	Share float64
}
var table = [2]entry{
	{Rate: 0.5 * 2.0, Share: 0.25 * 4.0},
	{Rate: 1.5 * 2.0, Share: 0.125 * 8.0},
}`,
		wantRules:          []string{},
		wantConstantExempt: 4,
	},
	{
		name: "accept/constant_product_whose_parent_is_an_addition",
		source: `package domain
const Total float64 = 0.5*2.0 + 3.0`,
		wantRules:          []string{},
		wantConstantExempt: 1,
	},
	{
		name: "reject/same_expression_with_one_runtime_operand",
		source: `package domain
func f(x float64) float64 { return 0.5*x + 3.0 }`,
		wantRules: []string{ruleProduct},
	},

	// ---- float-to-integer rule: rejection ----
	{
		name: "reject/int_conversion",
		source: `package domain
func f(x float64) int { return int(x) }`,
		wantRules: []string{ruleFloatInt},
	},
	{
		name: "reject/int64_conversion",
		source: `package domain
func f(x float64) int64 { return int64(x) }`,
		wantRules: []string{ruleFloatInt},
	},
	{
		name: "reject/uint32_conversion",
		source: `package domain
func f(x float64) uint32 { return uint32(x) }`,
		wantRules: []string{ruleFloatInt},
	},
	{
		name: "reject/named_integer_target",
		source: `package domain
type count int32
func f(x float64) count { return count(x) }`,
		wantRules: []string{ruleFloatInt},
	},
	{
		name: "reject/named_floating_source",
		source: `package domain
type ratio float64
func f(r ratio) int { return int(r) }`,
		wantRules: []string{ruleFloatInt},
	},
	{
		name: "reject/float32_source",
		source: `package domain
func f(x float32) int { return int(x) }`,
		wantRules: []string{ruleFloatInt},
	},
	{
		name: "reject/conversion_inside_a_test_file",
		path: "internal/domain/turn_test.go",
		source: `package domain
func check(x float64) int { return int(x) }`,
		wantRules: []string{ruleFloatInt},
	},

	// ---- float-to-integer rule: acceptance ----
	{
		name: "accept/integer_to_floating_conversion",
		source: `package domain
func f(i int) float64 { return float64(i) }`,
		wantRules: []string{},
	},
	{
		name: "accept/integer_to_integer_conversion",
		source: `package domain
func f(i int) int64 { return int64(i) }`,
		wantRules: []string{},
	},
	{
		name: "accept/constant_conversion_the_type_checker_folds",
		source: `package domain
const n = int(2.0)`,
		wantRules: []string{},
	},

	// ---- float-to-integer rule: the exemption is by file *and* name ----
	{
		name: "accept/round_population_is_the_one_audited_conversion",
		path: "internal/domain/quantities.go",
		source: `package domain
import "math"
type Population uint32
const MaxPopulation Population = 1<<32 - 1
func RoundPopulation(value float64) Population {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > float64(MaxPopulation) { return 0 }
	return Population(value + 0.5)
}`,
		wantRules: []string{},
	},
	{
		name: "accept/round_population_may_draw_from_an_rng",
		path: "internal/domain/quantities.go",
		source: `package domain
import "math"
type Population uint32
type WorldRNG struct{}
func (r *WorldRNG) Float64() float64 { return 0 }
const MaxPopulation Population = 1<<32 - 1
func RoundPopulation(value float64, rng *WorldRNG) Population {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > float64(MaxPopulation) { return 0 }
	return Population(value + rng.Float64())
}`,
		wantRules: []string{},
	},
	{
		name: "reject/round_population_with_an_rng_but_no_finite_checks",
		path: "internal/domain/quantities.go",
		source: `package domain
type Population uint32
type WorldRNG struct{}
func (r *WorldRNG) Float64() float64 { return 0 }
const MaxPopulation Population = 1<<32 - 1
func RoundPopulation(value float64, rng *WorldRNG) Population {
	if value < 0 || value > float64(MaxPopulation) { return 0 }
	return Population(value + rng.Float64())
}`,
		wantRules: []string{ruleFloatInt},
	},
	{
		name: "reject/round_population_with_two_floating_parameters",
		path: "internal/domain/quantities.go",
		source: `package domain
import "math"
type Population uint32
const MaxPopulation Population = 1<<32 - 1
func RoundPopulation(value float64, jitter float64) Population {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > float64(MaxPopulation) { return 0 }
	return Population(value + jitter)
}`,
		wantRules: []string{ruleFloatInt},
	},
	{
		name: "reject/round_population_without_finite_checks",
		path: "internal/domain/quantities.go",
		source: `package domain
type Population uint32
const MaxPopulation Population = 1<<32 - 1
func RoundPopulation(value float64) Population {
	if value < 0 || value > float64(MaxPopulation) { return 0 }
	return Population(value + 0.5)
}`,
		wantRules: []string{ruleFloatInt},
	},
	{
		name: "reject/package_scope_after_round_population",
		path: "internal/domain/quantities.go",
		source: `package domain
import "math"
type Population uint32
const MaxPopulation Population = 1<<32 - 1
func RoundPopulation(value float64) Population {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > float64(MaxPopulation) { return 0 }
	return Population(value + 0.5)
}
var source = float64(MaxPopulation)
var leaked = int(source)`,
		wantRules: []string{ruleFloatInt},
	},
	{
		name: "reject/another_function_in_quantities_go",
		path: "internal/domain/quantities.go",
		source: `package domain
type Population uint32
func roundOther(value float64) Population { return Population(value + 0.5) }`,
		wantRules: []string{ruleFloatInt},
	},
	{
		name: "reject/round_population_copied_into_another_file",
		path: "internal/domain/band.go",
		source: `package domain
type Population uint32
func RoundPopulation(value float64) Population { return Population(value + 0.5) }`,
		wantRules: []string{ruleFloatInt},
	},

	// ---- the rules are scoped to the domain ----
	{
		name: "accept/math_and_conversions_outside_the_domain",
		path: "pkg/render/map.go",
		source: `package domain
import "math"
func f(x float64) int { return int(math.Sqrt(x)) }`,
		wantRules: []string{},
	},
}

func TestArithmeticFixtures(t *testing.T) {
	for _, fixture := range arithmeticFixtures {
		t.Run(fixture.name, func(t *testing.T) {
			path := fixture.path
			if path == "" {
				path = "internal/domain/fixture.go"
			}
			report := analyzeFixture(t, path, fixture.source)
			if got := report.rules(); !slices.Equal(got, fixture.wantRules) {
				t.Fatalf("rules = %v, want %v\nreport:\n%s", got, fixture.wantRules, report)
			}
			if report.ConstantExempt != fixture.wantConstantExempt {
				t.Errorf("constant-folded products = %d, want %d", report.ConstantExempt, fixture.wantConstantExempt)
			}
			if report.ChainExempt != fixture.wantChainExempt {
				t.Errorf("chained products exempted = %d, want %d", report.ChainExempt, fixture.wantChainExempt)
			}
		})
	}
}
