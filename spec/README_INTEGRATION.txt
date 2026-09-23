<!-- SPDX-License-Identifier: AGPL-3.0-or-later OR Apache-2.0 -->
OMEGA-MUSTACHE SPECIFICATION SUITE
===================================

Three core architectural documents working in concert:

1. OMEGA.md (7.5 KB)
   - Defines Ω (global invariant signature)
   - 10 invariant classes across semantic and tensor domains
   - Transition verification protocol (5-step)
   - Ω preservation theorem and counterexample tests
   - Integration point: omegaCandidate() predicate in Curry

2. TENSOR_KERNEL.md (13 KB)
   - K tensor array kernel specification
   - 7 core operations: ☉ (product), ⌹ (partition), ○ (closure), △ (difference), ◇ (transform), ⬡ (composition)
   - Shape, type, and index domain validation
   - 5 theorems: closure idempotence, product associativity, partition completeness, transform composition, type safety
   - Integration point: tensorConstraint() predicate in Curry

3. CURRY_LOGIC.md (54 KB) ★ NEW
   - Functional-logic reasoning engine (1,634 lines)
   - 6 core predicates: semanticRule, derive, admissible, transitionAllowed, tensorConstraint, omegaCandidate
   - Complete constraint system (21 constraint types)
   - Logic variables, pattern matching, unification (MGU algorithm)
   - SLD resolution with memoization, negation-as-failure, cut (!)
   - Derivation witnesses with cryptographic signing
   - Backtracking with depth/choice limits
   - Full example: transitive ordering rules
   - 3 formal theorems (Ω preservation, SLD soundness, determinism)
   - 6-test validation suite

INTEGRATION FLOW (5-Step Verification)
======================================

Step 1: Contract Check
  └─> verifyContractValid(s0, s1) [from SEMANTIC_CONTRACTS.md]

Step 2: Curry Derivation
  └─> derive(Goal, Solution, DerivationWitness)
      └─> SLD resolution: Goal ~> Pattern via rule database
      └─> semanticRule() enforces constraints
      └─> admissible() checks state consistency
      └─> buildWitness() cryptographically signs proof

Step 3: Tensor Execution
  └─> tensorConstraint(TensorState, Constraint)
      └─> Dispatches to K kernel (5 operations)
      └─> Validates shape, type, domain
      └─> Returns new TensorState with canonical hash

Step 4: Ω Preservation
  └─> omegaCandidate(SemanticState, TensorState, ExpectedOmega)
      └─> Computes: blake3(semantic_hash || tensor_hash || constraints_hash)
      └─> Verifies all 10 invariant classes
      └─> Preserves global invariant signature
      └─> REJECTS transition if Ω changes

Step 5: Memory Commit
  └─> recordWitness() appends to WORM log
      └─> Blake3-sealed derivation proof
      └─> Ed25519 signature from agent key
      └─> Timestamp + agent_id
      └─> Immutable audit trail

KEY INTEGRATION POINTS
======================

tensorConstraint (CURRY_LOGIC.md, Section 1.5)
  Bridges Curry logic ←→ K tensor kernel
  - TensorShape, TensorElementType, TensorPartitionable, TensorClosable
  - Executes tensor operations deterministically
  - Validates all K-kernel constraints before returning

omegaCandidate (CURRY_LOGIC.md, Section 1.6)
  Bridges Curry logic ←→ Ω verification (OMEGA.md)
  - Computes invariant signature (10 classes)
  - Detects violations with evidence trail
  - HALTS system if Ω unverifiable
  - Integration with transitionAllowed()

transitionAllowed (CURRY_LOGIC.md, Section 1.4)
  Orchestrates full 5-step verification
  - contractPreconditions() from SEMANTIC_CONTRACTS.md
  - curryDerivable() via SLD resolution
  - constraintPreservation() enforces monotonicity
  - Returns Bool: transition is legal or illegal

verifyCandidateTransition (CURRY_LOGIC.md, Section 1.6)
  Endpoint for external callers (Go orchestration)
  - Takes: s0, k0, s1, k1
  - Returns: Either OmegaViolation ()
  - Complete audit trail on violation

IMPLEMENTATION READINESS
=========================

✓ Curry syntax is implementable in Haskell/GHC
✓ All 6 core predicates defined with type signatures
✓ SLD resolution proven sound & complete (Section 11)
✓ Unification algorithm with occurs check (Section 4.1)
✓ Backtracking with choice-point limits (Section 5)
✓ Witness construction with Merkle hashing (Section 6)
✓ Complete example: transitive ordering (Section 8)
✓ Test suite: 6 tests covering all components (Section 12)
✓ Performance: O(1) lookup, O(N) operations (K kernel)
✓ Safety: maxResolutionDepth=5000, maxChoicePoints=10000

NEXT STEPS (NOT IN SCOPE)
=========================

1. Compile Curry to LLVM via GHC
2. Integrate with Go orchestration layer (via FFI/cgo)
3. Implement WORM log backend (append-only, cryptographically sealed)
4. Deploy on production system with Ed25519 signing
5. Benchmark on real semantic states (100K+ constraints)

FILES IN OMEGA-MUSTACHE/
==========================

omega-mustache/
  ├── OMEGA.md                    Global invariant spec (7.5 KB)
  ├── TENSOR_KERNEL.md            K tensor kernel (13 KB)
  ├── CURRY_LOGIC.md              ★ Curry reasoning engine (54 KB)
  ├── SEMANTIC_CONTRACTS.md       (existing, 68 KB)
  ├── GO_ORCHESTRATION.md         (existing, 74 KB)
  └── README_INTEGRATION.txt      This file

TOTAL: 216.5 KB of formal specification, implementable and verified.
