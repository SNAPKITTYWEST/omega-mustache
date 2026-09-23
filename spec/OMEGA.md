<!-- SPDX-License-Identifier: AGPL-3.0-or-later OR Apache-2.0 -->
<!-- CLONE_GATE:AES256:f77ebc83d4ee51886072e270fbdcac1df96aac78cdcd2ad96a833a9e9f599fe2 -->

# Ω Global Invariant Specification

## 1. Ω Definition

```
Ω : (SemanticState × TensorState) → InvariantSignature
```

### Canonical Form

```prolog
omega(S, K, Ω_sig) :-
    state_canonical(S),
    tensor_canonical(K),
    compute_invariant_signature(S, K, Ω_sig).

% Ω_sig is a cryptographic hash representing the global invariant state
```

## 2. Primary Law

**No transition can modify Ω without violation.**

```
VALID_TRANSITION(S₀, K₀, S₁, K₁)

  ⟹

omega(S₀, K₀, Ω₀)
∧ omega(S₁, K₁, Ω₁)
∧ Ω₀ = Ω₁
```

### Proof Requirement

Before any state commit:

1. **Contract Check**: SemanticContract(S₀, S₁) succeeds
2. **Logic Derivation**: Curry produces valid derivation
3. **Tensor Operation**: K executes transformation (K₀ → K₁)
4. **Ω Verification**: omega(S₀, K₀, Ω₀) = omega(S₁, K₁, Ω₁)
5. **Memory Commit**: Record added to WORM chain

ALL FIVE must pass. Failure in ANY stage rejects the transition.

## 3. Components of Ω

### SemanticState Invariants

```prolog
semantic_invariant(S) :-
    % (1) Well-formedness
    well_formed_state(S),
    
    % (2) Consistency
    consistent_constraints(S),
    
    % (3) Admissibility
    admissible_under_contract(S),
    
    % (4) Determinism
    deterministic_resolution(S),
    
    % (5) Acyclicity (if applicable)
    acyclic_dependencies(S).
```

### TensorState Invariants

```prolog
tensor_invariant(K) :-
    % (1) Shape validity
    valid_shape(K),
    
    % (2) Element type consistency
    consistent_element_type(K),
    
    % (3) Index domain correctness
    valid_index_domain(K),
    
    % (4) Closure properties
    closure_preserved(K),
    
    % (5) Partition completeness (if applicable)
    partition_complete(K).
```

### Computed Signature

```prolog
invariant_signature(S, K, Hash) :-
    canonical_state_hash(S, S_hash),
    canonical_tensor_hash(K, K_hash),
    invariant_constraints_hash(S, K, C_hash),
    blake3(S_hash || K_hash || C_hash, Hash).
```

## 4. Transition Semantics

### Legal Transition

```
transition(S₀, K₀, S₁, K₁)

is legal iff:

(1) contract_valid(S₀, S₁)         [semantic preconditions]
(2) curry_derivation(S₀, S₁)       [logic derives conclusion]
(3) tensor_execution(K₀, K₁)       [tensor kernel executes]
(4) omega_preserved(S₀, K₀, S₁, K₁)    [Ω unchanged]
(5) memory_record(S₀, K₀, S₁, K₁)  [history appended]
```

### Illegal Transition

Any transition where:
- Contract preconditions violated → **REJECT**
- Curry derivation fails/ambiguous → **REJECT**
- Tensor operation invalid → **REJECT**
- Ω changes → **REJECT** (with evidence of violation)
- Memory commit fails → **REJECT** (atomic rollback)

## 5. Ω Preservation Proof

### Invariant Class

```prolog
invariant_class(Class) :-
    member(Class, [
        semantic_well_formedness,
        constraint_consistency,
        state_admissibility,
        deterministic_resolution,
        dependency_acyclicity,
        tensor_shape_validity,
        tensor_type_consistency,
        tensor_index_domain,
        tensor_closure_property,
        partition_completeness
    ]).

% All must remain invariant across transitions
```

### Per-Class Proof Obligation

For each invariant class C:

```prolog
proof_obligation(C, T1, T2) :-
    % T1 = before state, T2 = after state
    class_invariant(C, T1),
    transition(T1, T2),
    class_invariant(C, T2).
```

## 6. Ω as a Monad

```haskell
-- In Haskell/Curry pseudocode

newtype Ω a = Ω {
    runΩ :: SemanticState → TensorState → Either OmegaViolation (a, SemanticState, TensorState)
}

instance Monad Ω where
    return x = Ω $ \s k → Right (x, s, k)
    
    m >>= f = Ω $ \s k → do
        (a, s', k') <- runΩ m s k
        -- Check Ω preserved before continuing
        when (not (omega_equal (omega s k) (omega s' k'))) $
            Left OmegaViolated
        runΩ (f a) s' k'
```

## 7. Counterexample-Resistance

### Tests for Ω Stability

**Test 1: Empty Transition**
```
S₀ = S₁, K₀ = K₁ ⟹ Ω₀ = Ω₁ ✓
```

**Test 2: Contract-Valid Change**
```
contract_valid(S₀, S₁)
∧ S₀ ≠ S₁
⟹ Ω₀ = Ω₁ (not Ω₀ ≠ Ω₁)
```

**Test 3: Tensor Partition/Reconstruction**
```
K₀ = ⌹k(K₀)
∧ ○(⌹k(K₀)) = K₀
⟹ Ω(S, K₀) = Ω(S, K₀)
```

**Test 4: Concurrent Queries**
```
parallel(
  transition(S₀, K₀, S₁, K₁),
  transition(S₀, K₀, S₂, K₂)
)
⟹ exactly one succeeds (WORM enforces serialization)
```

**Test 5: Replay Determinism**
```
replay(history, t₀, t₁) = replay(history, t₀, t₁)
∧ omega(S, K) = omega(S', K') at same logical time
```

## 8. Ω Violations

### Violation Categories

| Category | Cause | Action |
|----------|-------|--------|
| **SEMANTIC_VIOLATED** | Contract preconditions failed | REJECT before commit |
| **LOGIC_VIOLATED** | Curry derivation failed/ambiguous | REJECT before commit |
| **TENSOR_VIOLATED** | K operation invalid | REJECT before commit |
| **OMEGA_VIOLATED** | Ω changed | REJECT with evidence + rollback |
| **MEMORY_VIOLATED** | WORM append failed | REJECT + emergency halt |

## 9. Ω Evidence Trail

When Ω is violated, produce:

```prolog
omega_violation_report(
    timestamp,
    agent_id,
    input_hash,
    semantic_before,
    semantic_after,
    tensor_before,
    tensor_after,
    omega_before,
    omega_after,
    violated_invariants,
    curry_derivation_(if available),
    tensor_operation_log,
    memory_state
).
```

## 10. Formalization in Curry

```curry
-- kernel/Omega.curry

omega :: SemanticState -> TensorState -> OmegaSignature
omega s k = blake3Signature (
    canonicalSemanticHash s ++
    canonicalTensorHash k ++
    invariantConstraintsHash s k
)

validTransition :: SemanticState -> TensorState 
                -> SemanticState -> TensorState 
                -> Bool
validTransition s0 k0 s1 k1 =
    contractValid s0 s1 &&
    curryDerivable s0 s1 &&
    tensorExecutable k0 k1 &&
    omega s0 k0 == omega s1 k1 &&
    memoryCommittable s0 k0 s1 k1

preserveOmega :: SemanticState -> TensorState 
              -> SemanticState -> TensorState 
              -> Either OmegaViolation ()
preserveOmega s0 k0 s1 k1
  | omega s0 k0 == omega s1 k1 = Right ()
  | otherwise = Left (OmegaViolation s0 k0 s1 k1)
```

## 11. Theorem: Ω Completeness

**Theorem**: If all component invariants are preserved, then Ω is preserved.

```prolog
theorem_omega_completeness :-
    forall(
        invariant_class(Class),
        (class_invariant(Class, S0, K0),
         transition(S0, K0, S1, K1),
         class_invariant(Class, S1, K1))
    )
    ->
    omega(S0, K0, O0),
    omega(S1, K1, O1),
    O0 = O1.

% Proof: Ω is a function of canonical state + all component invariants.
% If all components are preserved, their combination is preserved.
```

## 12. Ω Audit Trail

Every state transition generates:

```
ΩAuditRecord {
    record_id,
    timestamp,
    transition_id,
    agent_id,
    semantic_state_before_hash,
    semantic_state_after_hash,
    tensor_state_before_hash,
    tensor_state_after_hash,
    omega_before,
    omega_after,
    omega_preserved (bool),
    all_component_invariants_checked (bool),
    curry_derivation_hash,
    contract_version,
    previous_record_hash,
    record_hash
}
```

All records append to WORM. Ω verification must succeed before record is sealed.

## 13. Emergency Halt Condition

If at any point Ω cannot be verified:

```
OMEGA_UNVERIFIABLE
    ⟹
AGENT HALTS
MEMORY STATE PRESERVED
VIOLATION LOGGED
EXTERNAL NOTIFICATION SENT
```

**No recovery without manual intervention and external audit.**
