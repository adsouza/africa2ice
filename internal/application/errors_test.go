package application

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/internal/domain"
	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// domainErrorCases is the step 6 table fixture: every exported domain error
// with the stable gameapi code it maps to. The three invariant failures carry
// no player-actionable category, so they map to ErrInvalidCommand by design.
// The unnamed final row is an unclassified error, which must also land there.
var domainErrorCases = []struct {
	name string
	err  error
	code gameapi.ErrorCode
}{
	{"ErrBandNotFound", domain.ErrBandNotFound, gameapi.ErrBandNotFound},
	{"ErrComputerControlledBand", domain.ErrComputerControlledBand, gameapi.ErrComputerControlledBand},
	{"ErrInvalidAssignment", domain.ErrInvalidAssignment, gameapi.ErrInvalidAssignment},
	{"ErrSpatialActionUsed", domain.ErrSpatialActionUsed, gameapi.ErrSpatialActionUsed},
	{"ErrInvalidMigration", domain.ErrInvalidMigration, gameapi.ErrInvalidMigration},
	{"ErrSplitStressTooLow", domain.ErrSplitStressTooLow, gameapi.ErrSplitStressTooLow},
	{"ErrSplitPopulationTooLow", domain.ErrSplitPopulationTooLow, gameapi.ErrSplitPopulationTooLow},
	{"ErrSplitDestinationNotAdjacent", domain.ErrSplitDestinationNotAdjacent, gameapi.ErrSplitDestinationNotAdjacent},
	{"ErrSplitDestinationUninhabitable", domain.ErrSplitDestinationUninhabitable, gameapi.ErrSplitDestinationUninhabitable},
	{"ErrSplitDestinationUnexplored", domain.ErrSplitDestinationUnexplored, gameapi.ErrSplitDestinationUnexplored},
	{"ErrBandLimitReached", domain.ErrBandLimitReached, gameapi.ErrBandLimitReached},
	{"ErrBandIDExhausted", domain.ErrBandIDExhausted, gameapi.ErrBandIDExhausted},
	{"ErrMissingTechnologyPrerequisite", domain.ErrMissingTechnologyPrerequisite, gameapi.ErrMissingTechnologyPrerequisite},
	{"ErrTechnologyAlreadyAcquired", domain.ErrTechnologyAlreadyAcquired, gameapi.ErrTechnologyAlreadyAcquired},
	{"ErrInvalidInterbreedTarget", domain.ErrInvalidInterbreedTarget, gameapi.ErrInvalidInterbreedTarget},
	{"ErrCampaignComplete", domain.ErrCampaignComplete, gameapi.ErrCampaignComplete},
	{"ErrInvalidValue", domain.ErrInvalidValue, gameapi.ErrInvalidCommand},
	{"ErrInvalidTurn", domain.ErrInvalidTurn, gameapi.ErrInvalidCommand},
	{"ErrInvalidCoordinate", domain.ErrInvalidCoordinate, gameapi.ErrInvalidCommand},
	{"", errors.New("unexpected failure"), gameapi.ErrInvalidCommand},
}

// The application boundary must preserve refusal categories even when domain
// operations add context, so clients can select the correct recovery action.
func TestDomainErrorContract(t *testing.T) {
	for _, tc := range domainErrorCases {
		t.Run(tc.err.Error(), func(t *testing.T) {
			for _, err := range []error{tc.err, fmt.Errorf("command refused: %w", tc.err)} {
				got := mapDomainError(err)
				if got.Code != tc.code || got.Message != err.Error() {
					t.Fatalf("mapped error = %#v, want code %v and message %q", got, tc.code, err.Error())
				}
				if domainErrorCode(err) != got.Code {
					t.Fatal("projection and command error classification differ")
				}
			}
		})
	}
}

// The table above is only exhaustive if it is checked against the domain. This
// parses every non-test file in internal/domain for exported package-level
// Err* variables and requires a row for each, so a new domain error cannot
// silently fall through to ErrInvalidCommand without a deliberate entry.
func TestDomainErrorCasesCoverEveryExportedDomainError(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "domain", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	declared := map[string]bool{}
	fileSet := token.NewFileSet()
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			general, ok := decl.(*ast.GenDecl)
			if !ok || general.Tok != token.VAR {
				continue
			}
			for _, spec := range general.Specs {
				for _, name := range spec.(*ast.ValueSpec).Names {
					if strings.HasPrefix(name.Name, "Err") && name.IsExported() {
						declared[name.Name] = true
					}
				}
			}
		}
	}
	if len(declared) == 0 {
		t.Fatal("found no exported domain errors; the source walk is broken")
	}
	covered := map[string]bool{}
	for _, tc := range domainErrorCases {
		if tc.name != "" {
			covered[tc.name] = true
		}
	}
	for name := range declared {
		if !covered[name] {
			t.Errorf("domain.%s has no row in domainErrorCases", name)
		}
	}
	for name := range covered {
		if !declared[name] {
			t.Errorf("domainErrorCases names %s, which internal/domain no longer declares", name)
		}
	}
}
