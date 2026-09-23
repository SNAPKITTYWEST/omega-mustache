// SPDX-License-Identifier: AGPL-3.0-or-later OR Apache-2.0
// CLONE_GATE:AES256:f77ebc83d4ee51886072e270fbdcac1df96aac78cdcd2ad96a833a9e9f599fe2
package omega

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

// OmegaSignature is a Blake3-style hash representing the global invariant.
// Computed from canonical semantic + tensor state hashes.
type OmegaSignature string

// OmegaViolation is returned when a transition would change Ω.
type OmegaViolation struct {
	Before OmegaSignature
	After  OmegaSignature
	// Which of the 10 invariant classes failed
	ViolatedClasses []InvariantClass
}

func (v *OmegaViolation) Error() string {
	return fmt.Sprintf("Ω violated: before=%s after=%s classes=%v",
		v.Before, v.After, v.ViolatedClasses)
}

// InvariantClass is one of the 10 classes that compose Ω.
type InvariantClass string

const (
	SemanticWellFormedness InvariantClass = "semantic_well_formedness"
	ConstraintConsistency  InvariantClass = "constraint_consistency"
	StateAdmissibility     InvariantClass = "state_admissibility"
	DeterministicRes       InvariantClass = "deterministic_resolution"
	DependencyAcyclicity   InvariantClass = "dependency_acyclicity"
	TensorShapeValidity    InvariantClass = "tensor_shape_validity"
	TensorTypeConsistency  InvariantClass = "tensor_type_consistency"
	TensorIndexDomain      InvariantClass = "tensor_index_domain"
	TensorClosureProperty  InvariantClass = "tensor_closure_property"
	PartitionCompleteness  InvariantClass = "partition_completeness"
)

// AllInvariantClasses is the complete ordered set.
var AllInvariantClasses = []InvariantClass{
	SemanticWellFormedness,
	ConstraintConsistency,
	StateAdmissibility,
	DeterministicRes,
	DependencyAcyclicity,
	TensorShapeValidity,
	TensorTypeConsistency,
	TensorIndexDomain,
	TensorClosureProperty,
	PartitionCompleteness,
}

// Verifier checks Ω invariants and computes signatures.
type Verifier struct{}

// NewVerifier returns a ready Verifier.
func NewVerifier() *Verifier { return &Verifier{} }

// Compute derives the Ω signature from canonical hashes.
// Mirrors: blake3(S_hash || K_hash || C_hash)
func (v *Verifier) Compute(semanticHash, tensorHash, constraintHash string) OmegaSignature {
	combined := semanticHash + "|" + tensorHash + "|" + constraintHash
	h := sha256.Sum256([]byte(combined))
	return OmegaSignature(hex.EncodeToString(h[:]))
}

// Preserve verifies Ω is identical before and after a transition.
func (v *Verifier) Preserve(before, after OmegaSignature) error {
	if before != after {
		return &OmegaViolation{Before: before, After: after}
	}
	return nil
}

// CheckSemanticInvariants checks the 5 semantic component invariants.
// Returns the list of violated classes (empty = all pass).
func (v *Verifier) CheckSemanticInvariants(
	wellFormed bool,
	consistent bool,
	admissible bool,
	deterministic bool,
	acyclic bool,
) []InvariantClass {
	var failed []InvariantClass
	if !wellFormed {
		failed = append(failed, SemanticWellFormedness)
	}
	if !consistent {
		failed = append(failed, ConstraintConsistency)
	}
	if !admissible {
		failed = append(failed, StateAdmissibility)
	}
	if !deterministic {
		failed = append(failed, DeterministicRes)
	}
	if !acyclic {
		failed = append(failed, DependencyAcyclicity)
	}
	return failed
}

// CheckTensorInvariants checks the 5 tensor component invariants.
func (v *Verifier) CheckTensorInvariants(
	validShape bool,
	consistentType bool,
	validIndex bool,
	closurePreserved bool,
	partitionComplete bool,
) []InvariantClass {
	var failed []InvariantClass
	if !validShape {
		failed = append(failed, TensorShapeValidity)
	}
	if !consistentType {
		failed = append(failed, TensorTypeConsistency)
	}
	if !validIndex {
		failed = append(failed, TensorIndexDomain)
	}
	if !closurePreserved {
		failed = append(failed, TensorClosureProperty)
	}
	if !partitionComplete {
		failed = append(failed, PartitionCompleteness)
	}
	return failed
}

// ViolationReport is the evidence produced on Ω violation.
type ViolationReport struct {
	Timestamp          string
	AgentID            string
	InputHash          string
	SemanticBefore     string
	SemanticAfter      string
	TensorBefore       string
	TensorAfter        string
	OmegaBefore        OmegaSignature
	OmegaAfter         OmegaSignature
	ViolatedInvariants []InvariantClass
	CurryDerivation    string
	TensorOperationLog string
	MemoryState        string
}

// ErrOmegaUnverifiable is the emergency halt trigger.
var ErrOmegaUnverifiable = errors.New("OMEGA_UNVERIFIABLE: agent must halt")
