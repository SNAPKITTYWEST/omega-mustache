// SPDX-License-Identifier: AGPL-3.0-or-later OR Apache-2.0
// CLONE_GATE:AES256:e9dc53c70a432e67abd44f597d513fccc9359aa4f830137e6928f018f65b7450
package contract

import (
	"fmt"
	"strings"
)

// ContractID uniquely identifies a semantic contract.
type ContractID string

// ContractVersion is a semantic version string.
type ContractVersion string

// Evidence is a proof artifact produced during verification.
type Evidence struct {
	Stage   string
	Passed  bool
	Message string
}

// Contract is the 6-tuple (ID, Pre, Post, Invariants, Admissibility, Evidence).
type Contract struct {
	ID              ContractID
	Version         ContractVersion
	Preconditions   []Condition
	Postconditions  []Condition
	Invariants      []Condition
	AdmissibilityFn func(before, after State) bool
}

// Condition is a named predicate over a state pair.
type Condition struct {
	Name string
	Fn   func(before, after State) bool
}

// State is a typed semantic state snapshot.
// No map[string]interface{} — all fields are typed.
type State struct {
	AgentID         string
	OperationName   string
	InputHash       string
	OutputHash      string
	DependencyHashes []string
	Flags           StateFlags
	Payload         []byte
}

// StateFlags carries boolean state attributes.
type StateFlags struct {
	WellFormed    bool
	Consistent    bool
	Admissible    bool
	Deterministic bool
	Acyclic       bool
}

// VerificationResult is the output of the 8-phase pre-commitment pipeline.
type VerificationResult struct {
	ContractID ContractID
	Passed     bool
	Evidence   []Evidence
	FailPhase  int    // 0 if all passed
	FailReason string
}

// Evaluate runs the 8-phase verification pipeline for this contract.
//
// Phases:
//  1. Type check
//  2. State validation
//  3. Dependency resolution
//  4. Invariant check (before state)
//  5. Access control
//  6. Preconditions
//  7. Postconditions (simulated)
//  8. Admissibility
func (c *Contract) Evaluate(before, after State) VerificationResult {
	res := VerificationResult{ContractID: c.ID}

	phases := []struct {
		name string
		fn   func() (bool, string)
	}{
		{"type_check", func() (bool, string) {
			if before.AgentID == "" {
				return false, "agent_id must be non-empty"
			}
			if before.OperationName == "" {
				return false, "operation_name must be non-empty"
			}
			return true, "ok"
		}},
		{"state_validation", func() (bool, string) {
			if !before.Flags.WellFormed {
				return false, "before state not well-formed"
			}
			return true, "ok"
		}},
		{"dependency_resolution", func() (bool, string) {
			for _, dep := range before.DependencyHashes {
				if dep == "" {
					return false, "empty dependency hash"
				}
			}
			return true, "ok"
		}},
		{"invariant_before", func() (bool, string) {
			for _, inv := range c.Invariants {
				if !inv.Fn(before, before) {
					return false, fmt.Sprintf("invariant %q failed on before state", inv.Name)
				}
			}
			return true, "ok"
		}},
		{"access_control", func() (bool, string) {
			// Stub: real impl checks ACL table
			return true, "ok"
		}},
		{"preconditions", func() (bool, string) {
			for _, pre := range c.Preconditions {
				if !pre.Fn(before, after) {
					return false, fmt.Sprintf("precondition %q failed", pre.Name)
				}
			}
			return true, "ok"
		}},
		{"postconditions", func() (bool, string) {
			for _, post := range c.Postconditions {
				if !post.Fn(before, after) {
					return false, fmt.Sprintf("postcondition %q failed", post.Name)
				}
			}
			return true, "ok"
		}},
		{"admissibility", func() (bool, string) {
			if c.AdmissibilityFn != nil && !c.AdmissibilityFn(before, after) {
				return false, "admissibility check failed"
			}
			return true, "ok"
		}},
	}

	for i, phase := range phases {
		passed, msg := phase.fn()
		res.Evidence = append(res.Evidence, Evidence{
			Stage:   phase.name,
			Passed:  passed,
			Message: msg,
		})
		if !passed {
			res.Passed = false
			res.FailPhase = i + 1
			res.FailReason = fmt.Sprintf("phase %d (%s): %s", i+1, phase.name, msg)
			return res
		}
	}

	res.Passed = true
	return res
}

// Registry holds all registered contracts.
type Registry struct {
	contracts map[ContractID]*Contract
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{contracts: make(map[ContractID]*Contract)}
}

// Register adds a contract to the registry.
func (r *Registry) Register(c *Contract) error {
	if _, exists := r.contracts[c.ID]; exists {
		return fmt.Errorf("contract: %q already registered", c.ID)
	}
	r.contracts[c.ID] = c
	return nil
}

// Get retrieves a contract by ID.
func (r *Registry) Get(id ContractID) (*Contract, error) {
	c, ok := r.contracts[id]
	if !ok {
		return nil, fmt.Errorf("contract: %q not found", id)
	}
	return c, nil
}

// ViolationSummary formats evidence into a human-readable report.
func ViolationSummary(res VerificationResult) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Contract %s verification FAILED at phase %d: %s\n",
		res.ContractID, res.FailPhase, res.FailReason))
	for _, ev := range res.Evidence {
		mark := "✓"
		if !ev.Passed {
			mark = "✗"
		}
		sb.WriteString(fmt.Sprintf("  [%s] %s: %s\n", mark, ev.Stage, ev.Message))
	}
	return sb.String()
}
