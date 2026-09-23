<!-- SPDX-License-Identifier: AGPL-3.0-or-later OR Apache-2.0 -->
<!-- CLONE_GATE:AES256:e9dc53c70a432e67abd44f597d513fccc9359aa4f830137e6928f018f65b7450 -->

# SEMANTIC CONTRACTS Specification for OMEGA-MUSTACHE

**Version:** 1.0  
**Status:** Final Specification  
**Date:** 2026-09-22  
**Author:** Formal Methods — OMEGA-MUSTACHE Project  

---

## 1. Executive Summary

Semantic contracts are formal agreements that govern all state-changing operations in OMEGA-MUSTACHE. Each contract encodes:

- **What must be true before execution** (preconditions)
- **What must be true after execution** (postconditions)
- **What invariants are preserved** (loop/structural invariants)
- **When state transitions are permissible** (admissibility rules)
- **How violations are detected and rejected** (verification logic)

Contracts operate at three levels:

1. **Type-level contracts**: Method signatures, object allocation, field access
2. **Logic-level contracts**: State machines, Curry-logic propositions, Ω-verification lemmas
3. **Proof-level contracts**: Invariant preservation, bisimulation, semantic equivalence

All contracts are checked **before commitment**. Violations cause immediate rejection with proof of why the transition was illegal.

---

## 2. Contract Structure

### 2.1 Fundamental Contract Form

A semantic contract is a 6-tuple:

```
Contract ::= (ID, Preconditions, Postconditions, Invariants, Admissibility, Evidence)
```

**Components:**

| Component | Type | Purpose |
|-----------|------|---------|
| `ID` | UUID | Unique contract identifier for tracing and audit |
| `Preconditions` | `Set[Proposition]` | Conditions that must hold **before** the operation |
| `Postconditions` | `Set[Proposition]` | Conditions that must hold **after** successful operation |
| `Invariants` | `Set[Invariant]` | Structural constraints preserved across state transitions |
| `Admissibility` | `Rule` | Decision logic determining if transition is permitted |
| `Evidence` | `Proof` | Cryptographic commitment to contract satisfaction |

### 2.2 Precondition Taxonomy

Preconditions are grouped into five categories:

#### 2.2.1 Type Preconditions
Verify that arguments have correct types and kind signatures.

```prolog
type_precondition(Operation, Arg, Type) :-
    operation_signature(Operation, ArgTypes),
    nth0(ArgPosition, ArgTypes, Type),
    type_of(Arg, Type),
    \+ type_violation(Arg, Type).

type_violation(Arg, Type) :-
    type_of(Arg, ActualType),
    ActualType \= Type,
    \+ subtype(ActualType, Type).
```

**Example:** Method `push(x: Integer)` requires `x` to be integer-typed.

#### 2.2.2 State Validation Preconditions
Verify the runtime state is in a permissible configuration.

```prolog
state_valid(Operation, State) :-
    state_required(Operation, RequiredState),
    state_matches(State, RequiredState),
    \+ state_conflict(State, Operation).

state_matches(State, RequiredState) :-
    forall(
        state_property(RequiredState, Property, Value),
        (state_property(State, Property, V), V = Value)
    ).

state_conflict(State, Operation) :-
    forbidden_state_for(Operation, ForbiddenState),
    state_matches(State, ForbiddenState).
```

**Example:** `dequeue()` operation requires the queue state to be non-empty.

#### 2.2.3 Dependency Resolution Preconditions
Verify all dependencies are available and initialized.

```prolog
dependencies_satisfied(Operation, State) :-
    forall(
        required_dependency(Operation, Dep),
        (
            dependency_available(Dep, State),
            dependency_initialized(Dep, State)
        )
    ).

dependency_available(Dep, State) :-
    state_binding(State, Dep, _).

dependency_initialized(Dep, State) :-
    state_binding(State, Dep, Binding),
    binding_status(Binding, initialized).
```

**Example:** Object method call requires the object to be non-null and fully constructed.

#### 2.2.4 Invariant Preconditions
Verify that all class invariants hold in the pre-state.

```prolog
invariants_hold_pre(Operation, State) :-
    forall(
        class_invariant(Operation, Invariant),
        evaluate_invariant(Invariant, State)
    ).

evaluate_invariant(Inv, State) :-
    Inv =.. [Predicate | Args],
    call(Predicate, State, Args).
```

**Example:** For a sorted list: `is_sorted(list)` must be true before any insertion.

#### 2.2.5 Access Control Preconditions
Verify caller has permission to execute the operation.

```prolog
access_permitted(Operation, Caller, State) :-
    operation_visibility(Operation, Visibility),
    caller_clearance(Caller, Clearance),
    visibility_check(Visibility, Clearance),
    \+ access_denied(Caller, Operation, State).

visibility_check(public, _).
visibility_check(protected, object_member_or_subclass).
visibility_check(private, exact_object_member).
```

---

### 2.3 Postcondition Taxonomy

Postconditions are grouped into five categories:

#### 2.3.1 State Validity Postconditions
Verify the result state is well-formed and consistent.

```prolog
result_state_valid(Operation, PreState, PostState) :-
    state_structurally_valid(PostState),
    state_semantically_valid(PostState),
    causality_respected(PreState, PostState, Operation).

state_structurally_valid(State) :-
    \+ null_pointer_violation(State),
    \+ memory_corruption(State),
    \+ type_mismatch_in_bindings(State).

state_semantically_valid(State) :-
    all_invariants_hold(State),
    no_unreachable_bindings(State).

causality_respected(PreState, PostState, Operation) :-
    forall(
        state_binding(PreState, Var, Binding),
        (
            \+ affected_by(Var, Operation)
            -> state_binding(PostState, Var, Binding)
            ;  true
        )
    ).
```

**Example:** After `append(list1, list2)`, the result is a valid list with correct length.

#### 2.3.2 Invariant Preservation Postconditions
Verify that all class and logical invariants are preserved.

```prolog
invariants_preserved(Operation, PreState, PostState) :-
    forall(
        class_invariant(Operation, Inv),
        (
            evaluate_invariant(Inv, PreState),
            evaluate_invariant(Inv, PostState)
        )
    ).

evaluate_invariant(Inv, State) :-
    Inv =.. [Pred | Args],
    call(Pred, State, Args).
```

**Example:** After sorting: `is_sorted(list)` must still be true.

#### 2.3.3 Effect Recording Postconditions
Verify that intended side effects are recorded correctly.

```prolog
effects_recorded(Operation, PreState, PostState, EffectLog) :-
    forall(
        intended_effect(Operation, Effect),
        recorded_effect(Effect, EffectLog),
        effect_matches(Effect, EffectLog)
    ).

recorded_effect(Effect, EffectLog) :-
    member(LogEntry, EffectLog),
    log_effect(LogEntry, Effect).

effect_matches(Effect, EffectLog) :-
    findall(Entry, log_effect(Entry, Effect), Entries),
    length(Entries, Count),
    Count > 0.
```

**Example:** After `write(file, data)`, the write is recorded in the effect log with timestamp and data hash.

#### 2.3.4 Output Validity Postconditions
Verify that computed outputs satisfy expected properties.

```prolog
output_valid(Operation, Outputs, PostState) :-
    forall(
        output_specification(Operation, OutputSpec),
        output_satisfies_spec(Outputs, OutputSpec, PostState)
    ).

output_satisfies_spec(Outputs, OutputSpec, State) :-
    OutputSpec = output(Type, Properties),
    type_of(Outputs, Type),
    forall(
        member(Prop, Properties),
        evaluate_property(Outputs, Prop, State)
    ).

evaluate_property(Value, Prop, State) :-
    Prop =.. [PredName | Args],
    call(PredName, Value, State, Args).
```

**Example:** After `find(key)`, returned value (if non-null) matches the key and exists in the collection.

#### 2.3.5 Monotonicity/Ordering Postconditions
Verify ordering and causality constraints.

```prolog
ordering_respected(Operation, PreState, PostState, GlobalOrder) :-
    operation_timestamp(Operation, Timestamp),
    append_to_causal_order(GlobalOrder, Timestamp, Operation),
    no_causality_violation(PreState, PostState, GlobalOrder).

no_causality_violation(PreState, PostState, GlobalOrder) :-
    \+ (
        depends_on(A, B, GlobalOrder),
        executed_after(B, A, GlobalOrder),
        A \= B
    ).
```

---

### 2.4 Invariant Classification

#### 2.4.1 Structural Invariants
Constraints on object structure and relationships.

```prolog
structural_invariant(Inv) :-
    Inv = inv(Name, Predicate, Scope),
    scope_type(Scope, structural).

% Examples:
class_invariant(Queue, inv(queue_structure, queue_valid, class)) :-
    queue_valid(Q) :- 
        is_list(Q),
        length(Q, Len),
        Len >= 0.

class_invariant(BinaryTree, inv(bst_property, bst_valid, class)) :-
    bst_valid(Node) :-
        (
            Node = nil
            ; (
                Node = node(Left, Val, Right),
                is_integer(Val),
                all_less_than(Left, Val),
                all_greater_than(Right, Val),
                bst_valid(Left),
                bst_valid(Right)
            )
        ).
```

#### 2.4.2 Data Invariants
Constraints on data relationships and consistency.

```prolog
data_invariant(Inv) :-
    Inv = inv(Name, Predicate, Scope),
    scope_type(Scope, data).

% Examples:
data_invariant(HashMap, inv(hash_consistency, hash_consistent, class)) :-
    hash_consistent(Map) :-
        forall(
            member(key(K) = V, Map),
            hash_lookup(Map, K, V)
        ).

data_invariant(LinkedList, inv(link_integrity, links_valid, class)) :-
    links_valid(Node) :-
        (
            Node = nil
            ; (
                Node = cell(Data, Next),
                (Next = nil ; is_node(Next)),
                \+ cyclic_link(Node)
            )
        ).
```

#### 2.4.3 Logical Invariants
Constraints derived from formal logic (Curry logic, Ω-verification).

```prolog
logical_invariant(Inv) :-
    Inv = inv(Name, Proposition, Scope),
    scope_type(Scope, logical).

% Examples:
logical_invariant(Transaction, inv(atomicity, atomic_commitment, global)) :-
    atomic_commitment(Tx) :-
        \+ (
            started(Tx),
            \+ (committed(Tx) ; aborted(Tx))
        ).

logical_invariant(Consensus, inv(safety, byzantine_safety, global)) :-
    byzantine_safety(State) :-
        forall(
            honest_node(N1),
            forall(
                honest_node(N2),
                view_agreement(N1, N2, State)
            )
        ).
```

#### 2.4.4 Security Invariants
Constraints on security properties (information flow, access control, crypto).

```prolog
security_invariant(Inv) :-
    Inv = inv(Name, Predicate, Scope),
    scope_type(Scope, security).

% Examples:
security_invariant(Encryption, inv(confidentiality, ciphertext_indistinguishable, global)) :-
    ciphertext_indistinguishable(Message) :-
        \+ observably_correlated(enc(Message, Key), enc(RandomBits, Key)).

security_invariant(MAC, inv(authenticity, signature_valid, operation)) :-
    signature_valid(Message, Signature, PublicKey) :-
        verify(Signature, Message, PublicKey).
```

---

## 3. State Transition Semantics

### 3.1 Transition Relation

A state transition is formalized as:

```
σ →_op σ'

Where:
  σ = pre-state
  σ' = post-state
  op = operation (method call, assignment, control flow)
```

**Formal definition:**

```prolog
state_transition(PreState, Operation, PostState) :-
    preconditions_hold(Operation, PreState),
    execution_computes(Operation, PreState, PostState),
    postconditions_hold(Operation, PreState, PostState),
    invariants_preserved(Operation, PreState, PostState),
    effect_recorded(Operation, PostState).
```

### 3.2 Transition Classes

#### 3.2.1 Deterministic Transitions
Single, well-defined outcome.

```prolog
deterministic_transition(Op, PreState, PostState) :-
    execution_deterministic(Op, PreState),
    \+ (
        state_transition(PreState, Op, PostState),
        state_transition(PreState, Op, PostState2),
        PostState \= PostState2
    ).
```

**Example:** `x := 42` always produces exactly one post-state.

#### 3.2.2 Nondeterministic Transitions
Multiple legal outcomes (choice, concurrency).

```prolog
nondeterministic_transition(Op, PreState, PostStates) :-
    findall(S, state_transition(PreState, Op, S), PostStates),
    length(PostStates, Count),
    Count > 1,
    forall(member(S, PostStates), admissible(S, Op)).

admissible(State, Op) :-
    postconditions_hold(Op, _, State),
    all_invariants_hold(State).
```

**Example:** `choose(x, {1, 2, 3})` produces three possible post-states.

#### 3.2.3 Guarded Transitions
Transitions that only occur when guards are satisfied.

```prolog
guarded_transition(Op, PreState, PostState, Guard) :-
    evaluate_guard(Guard, PreState),
    state_transition(PreState, Op, PostState).

evaluate_guard(Guard, State) :-
    Guard =.. [Pred | Args],
    call(Pred, State, Args).
```

**Example:** `if (x > 0) then y := x + 1` has guard `x > 0`.

#### 3.2.4 Conditional Transitions
Transitions with multiple guards (if-then-else).

```prolog
conditional_transition(Op, PreState, PostState) :-
    (
        evaluate_guard(Guard1, PreState)
        -> state_transition(PreState, Branch1, PostState)
        ; (
            evaluate_guard(Guard2, PreState)
            -> state_transition(PreState, Branch2, PostState)
            ; state_transition(PreState, DefaultBranch, PostState)
          )
    ).
```

---

### 3.3 Transition Sequences

A program execution is a sequence of transitions:

```
σ₀ →_op₁ σ₁ →_op₂ σ₂ ... →_opₙ σₙ
```

```prolog
execution_sequence(States, Operations) :-
    States = [StartState | Rest],
    Operations = [Op1, Op2, ...],
    execution_from(StartState, Operations, Rest).

execution_from(_, [], []).
execution_from(State, [Op | Ops], [NextState | NextStates]) :-
    state_transition(State, Op, NextState),
    execution_from(NextState, Ops, NextStates).

valid_execution(States, Operations) :-
    execution_sequence(States, Operations),
    forall(
        append(Prefix, [_Op | _], Operations),
        all_invariants_hold(element(length(Prefix) + 1, States))
    ).
```

---

## 4. Admissibility Rules

Admissibility rules determine whether a proposed state transition is legally permissible.

### 4.1 Admissibility Framework

```prolog
admissible_transition(PreState, Operation, PostState) :-
    % All preconditions satisfied
    preconditions_hold(Operation, PreState),
    % All postconditions satisfied
    postconditions_hold(Operation, PreState, PostState),
    % All invariants preserved
    invariants_preserved(Operation, PreState, PostState),
    % No security or safety violations
    no_violations(Operation, PreState, PostState),
    % Operation permitted by access control
    access_permitted(Operation, caller(PreState), PreState),
    % No conflicts with external constraints
    no_external_conflicts(Operation, PreState).
```

### 4.2 Precondition Satisfaction

```prolog
preconditions_hold(Operation, State) :-
    forall(
        precondition_of(Operation, Precond),
        evaluate_condition(Precond, State)
    ).

evaluate_condition(Cond, State) :-
    Cond =.. [Pred | Args],
    call(Pred, State, Args).
```

### 4.3 Invariant Preservation

```prolog
invariants_preserved(Operation, PreState, PostState) :-
    % Structural invariants
    (
        PreState satisfies structural_invariants
        -> PostState satisfies structural_invariants
    ),
    % Data invariants
    (
        PreState satisfies data_invariants
        -> PostState satisfies data_invariants
    ),
    % Logical invariants
    (
        PreState satisfies logical_invariants
        -> PostState satisfies logical_invariants
    ),
    % Security invariants
    (
        PreState satisfies security_invariants
        -> PostState satisfies security_invariants
    ).
```

### 4.4 Violation Detection

```prolog
no_violations(Operation, PreState, PostState) :-
    \+ type_violation(PreState, PostState, Operation),
    \+ null_pointer_violation(PreState, PostState),
    \+ memory_violation(PreState, PostState),
    \+ concurrency_violation(PreState, PostState, Operation),
    \+ information_flow_violation(PreState, PostState, Operation),
    \+ deadlock_violation(PreState, PostState, Operation).

type_violation(PreState, PostState, Operation) :-
    forall(
        binding(Var, Type) in PostState,
        type_of(Var, Type)
    ) -> false ; true.

null_pointer_violation(PreState, PostState) :-
    member(binding(Var, Value), PostState),
    Value = null,
    required_non_null(Var, PostState).

memory_violation(PreState, PostState) :-
    (
        allocated_beyond_heap(PostState)
        ; freed_use_after_free(PreState, PostState)
        ; double_free(PostState)
    ).

concurrency_violation(PreState, PostState, Operation) :-
    concurrent_operations_conflict(Operation, PostState),
    \+ mutual_exclusion_held(Operation, PostState).

information_flow_violation(PreState, PostState, Operation) :-
    information_leaks(PreState, PostState),
    security_level(Operation, Level),
    leaked_information_exceeds_level(PostState, Level).

deadlock_violation(PreState, PostState, Operation) :-
    creates_circular_wait(Operation, PostState),
    holds_mutual_exclusion(Operation, PostState).
```

### 4.5 Conflict Detection

```prolog
no_external_conflicts(Operation, PreState) :-
    \+ operation_disabled(Operation, PreState),
    \+ conflicts_with_scheduler(Operation, PreState),
    \+ violates_global_constraints(Operation, PreState),
    \+ race_condition_exists(Operation, PreState).

operation_disabled(Operation, State) :-
    state_binding(State, disabled_ops, DisabledSet),
    member(Operation, DisabledSet).

conflicts_with_scheduler(Operation, State) :-
    state_binding(State, current_thread, Thread),
    operation_thread_restriction(Operation, AllowedThread),
    Thread \= AllowedThread.

violates_global_constraints(Operation, State) :-
    forall(
        global_constraint(Constraint),
        \+ constraint_satisfied(Constraint, State, Operation)
    ).

race_condition_exists(Operation, State) :-
    findall(Op2, concurrent_operation(Op2, State), ConcurrentOps),
    member(Op2, ConcurrentOps),
    (
        read_write_conflict(Operation, Op2, State)
        ; write_write_conflict(Operation, Op2, State)
    ),
    \+ ordered_by_sync(Operation, Op2, State).

read_write_conflict(Op1, Op2, State) :-
    reads(Op1, Var),
    writes(Op2, Var),
    last_write_before(Op2, Op1, State).

write_write_conflict(Op1, Op2, State) :-
    writes(Op1, Var),
    writes(Op2, Var),
    \+ strictly_ordered(Op1, Op2, State).
```

---

## 5. Contract Types

### 5.1 Method Call Contracts

```prolog
contract_type(method_call, MethodCallContract) :-
    MethodCallContract = contract(
        method_call,
        [
            % Preconditions
            receiver_non_null,
            receiver_initialized,
            parameters_type_valid,
            parameters_state_valid,
            method_visibility_satisfied,
            method_enabled_in_current_state,
            dependencies_available
        ],
        [
            % Postconditions
            return_value_valid,
            object_state_consistent,
            side_effects_recorded,
            invariants_preserved,
            no_resource_leaks
        ],
        [
            % Invariants
            inv(receiver_consistency, receiver_post_non_null),
            inv(side_effect_safety, all_side_effects_reversible),
            inv(resource_safety, allocated_resources_released_on_failure)
        ],
        % Admissibility
        admissible_method_call,
        % Evidence
        proof_of_satisfaction
    ).

admissible_method_call(Object, Method, Args, PreState) :-
    receiver_valid(Object, PreState),
    method_exists(Object, Method),
    parameter_arity_matches(Method, Args),
    forall(
        nth0(I, Args, Arg),
        (
            method_parameter_type(Method, I, Type),
            type_of(Arg, Type)
        )
    ),
    method_preconditions(Object, Method, Args, PreState),
    access_check(caller(PreState), Object, Method, PreState).

receiver_valid(Object, State) :-
    \+ null_or_undefined(Object),
    state_binding(State, Object, ObjectRecord),
    \+ partially_initialized(ObjectRecord),
    class_of(Object, Class),
    State contains invariants_for(Class).

method_exists(Object, Method) :-
    class_of(Object, Class),
    class_method(Class, Method).

parameter_arity_matches(Method, Args) :-
    method_signature(Method, Signature),
    Signature = signature(_, ParamTypes, _),
    length(ParamTypes, ParamCount),
    length(Args, ParamCount).

access_check(Caller, Object, Method, State) :-
    method_visibility(Object, Method, Visibility),
    (
        Visibility = public
        ; (Visibility = protected, is_subclass(caller_class(Caller), class_of(Object)))
        ; (Visibility = private, Caller = Object)
    ).
```

**Example:**

```prolog
% Contract: Queue.enqueue(x: Integer)
method_contract(Queue, enqueue, [X]) :-
    Preconditions = [
        queue_non_null(Queue),
        integer_typed(X),
        queue_not_full(Queue)  % if bounded
    ],
    Postconditions = [
        queue_size_incremented(Queue),
        element_at_tail(Queue, X),
        all_previous_elements_preserved(Queue)
    ],
    Invariants = [
        inv(queue_structure, is_list, _),
        inv(fifo_property, fifo_order_preserved, _)
    ].
```

### 5.2 Object Allocation Contracts

```prolog
contract_type(object_allocation, AllocationContract) :-
    AllocationContract = contract(
        object_allocation,
        [
            % Preconditions
            class_defined,
            heap_space_available,
            constructor_parameters_valid,
            constructor_preconditions_satisfied
        ],
        [
            % Postconditions
            object_allocated_on_heap,
            object_fully_initialized,
            all_fields_initialized,
            object_id_unique,
            object_added_to_reachability_set
        ],
        [
            % Invariants
            inv(object_uniqueness, no_duplicate_objects),
            inv(initialization_completeness, all_fields_initialized),
            inv(heap_integrity, heap_consistent)
        ],
        % Admissibility
        admissible_allocation,
        % Evidence
        proof_of_allocation
    ).

admissible_allocation(Class, Args, PreState, PostState) :-
    class_defined(Class),
    heap_has_space(PostState),
    \+ heap_fragmented(PostState),
    constructor_arity_matches(Class, Args),
    forall(
        nth0(I, Args, Arg),
        (
            constructor_parameter_type(Class, I, Type),
            type_of(Arg, Type)
        )
    ),
    constructor_preconditions(Class, Args, PreState),
    all_superclass_constructors_called(Class, PostState).

heap_has_space(State) :-
    state_binding(State, heap_total, Total),
    state_binding(State, heap_used, Used),
    Required is object_size(Class),
    Used + Required =< Total.

all_superclass_constructors_called(Class, State) :-
    forall(
        superclass(Class, Super),
        (
            state_binding(State, super_initialized, InitSet),
            member(Super, InitSet)
        )
    ).
```

**Example:**

```prolog
% Contract: new Node(value: Integer, next: Node)
allocation_contract(Node, [Value, Next]) :-
    Preconditions = [
        class_defined(Node),
        integer_typed(Value),
        (Next = null ; node_typed(Next)),
        heap_available(16)  % assuming 16 bytes
    ],
    Postconditions = [
        node_allocated_on_heap,
        node_value_set_to(Value),
        node_next_set_to(Next),
        node_id_fresh,
        node_in_reachability_set
    ],
    Invariants = [
        inv(node_structure, node_valid),
        inv(link_integrity, next_link_valid)
    ].
```

### 5.3 Field Update Contracts

```prolog
contract_type(field_update, FieldUpdateContract) :-
    FieldUpdateContract = contract(
        field_update,
        [
            % Preconditions
            object_non_null,
            object_initialized,
            field_exists,
            new_value_type_valid,
            field_write_permission,
            no_field_locks
        ],
        [
            % Postconditions
            field_value_updated,
            object_state_consistent,
            dependent_invariants_preserved,
            field_history_recorded
        ],
        [
            % Invariants
            inv(field_consistency, field_value_matches),
            inv(type_safety, field_type_preserved)
        ],
        % Admissibility
        admissible_field_update,
        % Evidence
        proof_of_field_update
    ).

admissible_field_update(Object, Field, NewValue, PreState, PostState) :-
    \+ null_or_undefined(Object),
    object_fully_initialized(Object, PreState),
    class_has_field(class_of(Object), Field),
    field_type(class_of(Object), Field, Type),
    type_of(NewValue, Type),
    field_is_writable(Object, Field, PreState),
    \+ field_locked(Object, Field, PreState),
    field_update_preserves_invariants(Object, Field, NewValue, PreState),
    record_field_change(Object, Field, PreState, PostState).

field_update_preserves_invariants(Object, Field, NewValue, State) :-
    forall(
        depends_on_field(Invariant, Field),
        (
            evaluate_invariant(Invariant, State),
            evaluate_invariant_with_field_change(Invariant, Object, Field, NewValue, State)
        )
    ).

record_field_change(Object, Field, PreState, PostState) :-
    get_field_value(Object, Field, PreState, OldValue),
    timestamp_now(Timestamp),
    create_field_change_record(Object, Field, OldValue, NewValue, Timestamp, Record),
    append_to_field_history(Object, Field, Record, PostState).
```

**Example:**

```prolog
% Contract: node.next := newNode
field_update_contract(Node, next, NewNode) :-
    Preconditions = [
        node_non_null(Node),
        node_initialized(Node),
        (NewNode = null ; node_typed(NewNode)),
        \+ cyclic_reference(Node, NewNode)
    ],
    Postconditions = [
        node_next_equals(Node, NewNode),
        old_links_preserved,
        link_history_recorded
    ],
    Invariants = [
        inv(link_integrity, no_cycles),
        inv(reference_validity, pointed_node_exists)
    ].
```

### 5.4 Control Flow Contracts

```prolog
contract_type(control_flow, ControlFlowContract) :-
    ControlFlowContract = contract(
        control_flow,
        [
            % Preconditions
            guard_evaluation_total,
            all_paths_reachable,
            no_unreachable_code
        ],
        [
            % Postconditions
            exactly_one_branch_taken,
            state_consistency_across_branches,
            loop_termination_guaranteed
        ],
        [
            % Invariants
            inv(determinism, single_execution_path),
            inv(progress, no_infinite_loops_on_valid_input),
            inv(consistency, all_branches_agree_on_invariants)
        ],
        % Admissibility
        admissible_control_flow,
        % Evidence
        proof_of_control_flow
    ).

admissible_control_flow(Branch, PreState) :-
    (
        Branch = if_then_else(Guard, ThenBranch, ElseBranch),
        total_guard_evaluation(Guard, PreState),
        (
            evaluate_guard(Guard, PreState)
            -> admissible_transition(PreState, ThenBranch, _)
            ;  admissible_transition(PreState, ElseBranch, _)
        )
    ; (
        Branch = while_loop(Guard, Body),
        total_guard_evaluation(Guard, PreState),
        loop_invariant_holds(Body, PreState),
        loop_terminates(Guard, Body, PreState)
    ) ; (
        Branch = for_loop(Init, Guard, Update, Body),
        execution_sequence_valid([Init], PreState, _),
        total_guard_evaluation(Guard, PreState),
        loop_terminates(Guard, Body, PreState)
    ) ; (
        Branch = switch(Discriminant, Cases),
        type_of(Discriminant, DiscType),
        all_cases_covered(DiscType, Cases)
    )
    ).

loop_invariant_holds(Body, State) :-
    forall(
        loop_invariant_of(Body, Inv),
        evaluate_invariant(Inv, State)
    ).

loop_terminates(Guard, Body, PreState) :-
    % Must show termination metric (fuel, rank, variant function)
    exists_decreasing_metric(Guard, Body, PreState),
    metric_non_negative(PreState),
    metric_decreases_on_iteration(Guard, Body, PreState).
```

---

## 6. Verification Logic

### 6.1 Pre-Commitment Verification

All contracts are verified before any state changes commit.

```prolog
verify_and_commit(Operation, PreState, PostState) :-
    % Phase 1: Structural verification
    verify_contract_structure(Operation, PreState),
    % Phase 2: Precondition verification
    verify_preconditions(Operation, PreState),
    % Phase 3: Execution verification
    verify_execution_sound(Operation, PreState, PostState),
    % Phase 4: Postcondition verification
    verify_postconditions(Operation, PreState, PostState),
    % Phase 5: Invariant verification
    verify_invariants(Operation, PreState, PostState),
    % Phase 6: Admissibility verification
    verify_admissible(Operation, PreState, PostState),
    % Phase 7: Proof generation
    generate_proof_commitment(Operation, PreState, PostState, Proof),
    % Phase 8: Atomic commit
    commit_atomic(PostState, Proof).

verify_contract_structure(Operation, State) :-
    contract_for(Operation, Contract),
    Contract = contract(_, PreConds, PostConds, Invs, AdmRule, _),
    is_list(PreConds),
    is_list(PostConds),
    is_list(Invs),
    callable(AdmRule).

verify_preconditions(Operation, State) :-
    contract_for(Operation, Contract),
    Contract = contract(_, PreConds, _, _, _, _),
    forall(
        member(PreCond, PreConds),
        (
            precondition_holds(PreCond, State),
            record_verification(precondition, PreCond, satisfied)
        )
    ).

verify_execution_sound(Operation, PreState, PostState) :-
    execution_trace(Operation, PreState, Trace),
    \+ (
        member(Step, Trace),
        undefined_behavior(Step, PreState)
    ),
    trace_deterministic(Operation, PreState)
        -> single_poststate(PostState)
        ;  poststate_in_admissible_set(PostState, Operation, PreState).

verify_postconditions(Operation, PreState, PostState) :-
    contract_for(Operation, Contract),
    Contract = contract(_, _, PostConds, _, _, _),
    forall(
        member(PostCond, PostConds),
        (
            postcondition_holds(PostCond, PreState, PostState),
            record_verification(postcondition, PostCond, satisfied)
        )
    ).

verify_invariants(Operation, PreState, PostState) :-
    contract_for(Operation, Contract),
    Contract = contract(_, _, _, Invariants, _, _),
    forall(
        member(Invariant, Invariants),
        (
            Invariant = inv(Name, Pred, Scope),
            invariant_holds(Pred, PreState),
            invariant_holds(Pred, PostState),
            record_verification(invariant, Invariant, preserved)
        )
    ).

verify_admissible(Operation, PreState, PostState) :-
    contract_for(Operation, Contract),
    Contract = contract(_, _, _, _, AdmRule, _),
    call(AdmRule, PreState, Operation, PostState),
    \+ violation_exists(PreState, PostState, Operation),
    record_verification(admissibility, Operation, admissible).

generate_proof_commitment(Operation, PreState, PostState, Proof) :-
    contract_for(Operation, Contract),
    Contract = contract(ID, _, _, _, _, _),
    % Cryptographically sign verification
    verification_data(Operation, PreState, PostState, Data),
    blake3_hash(Data, Hash),
    ed25519_sign(Hash, PrivateKey, Signature),
    Proof = proof(ID, Hash, Signature, timestamp_now).

commit_atomic(PostState, Proof) :-
    % Atomic commitment: update state and record proof in WORM log
    (
        update_state_atomic(PostState),
        record_proof_worm(Proof)
    )
    -> state_committed(PostState)
    ;  (rollback_state, fail).
```

### 6.2 Type Checking

```prolog
verify_type_safety(PreState, Operation, PostState) :-
    % Check all type preconditions
    forall(
        type_binding(Var, Type) in Operation arguments,
        type_of(Var, Type)
    ),
    % Check all intermediate type invariants
    \+ type_mismatch_in_execution_trace(Operation, PreState),
    % Check all type postconditions
    forall(
        type_binding(Var, Type) in Operation results,
        type_of(result_of(Operation), Type)
    ).

type_mismatch_in_execution_trace(Operation, State) :-
    execution_trace(Operation, State, Trace),
    member(Step, Trace),
    (
        Step = assignment(Var, Value),
        declared_type(Var, Type),
        \+ type_of(Value, Type)
    ; (
        Step = method_call(Recv, Method, Args),
        \+ method_signature_matches(Recv, Method, Args)
    ) ; (
        Step = array_access(Arr, Index),
        \+ (index_type_valid(Index), array_element_type(Arr, _))
    )
    ).
```

### 6.3 State Consistency Checking

```prolog
verify_state_consistency(State) :-
    % Check no dangling pointers
    \+ dangling_reference_exists(State),
    % Check no type inconsistencies
    \+ type_inconsistency_exists(State),
    % Check all invariants hold
    forall(
        class_invariant(_, Inv),
        evaluate_invariant(Inv, State)
    ),
    % Check no memory corruption
    \+ memory_corruption_exists(State),
    % Check reachability set is correct
    reachability_set_correct(State),
    % Check garbage collection markers valid
    gc_marks_consistent(State).

dangling_reference_exists(State) :-
    state_binding(State, Var, Value),
    Value \= null,
    \+ object_exists_in_heap(Value, State).

reachability_set_correct(State) :-
    compute_reachable_set(State, ComputedSet),
    state_binding(State, reachable_objects, StoredSet),
    ComputedSet = StoredSet.

gc_marks_consistent(State) :-
    forall(
        object_in_heap(Obj, State),
        (
            in_reachable_set(Obj, State) -> \+ marked_for_collection(Obj, State)
            ; (gc_graph_rooted(State) -> marked_for_collection(Obj, State))
        )
    ).
```

### 6.4 Curry Logic Integration

Contracts can express logical constraints using Curry logic.

```prolog
curry_precondition(Operation, Proposition) :-
    % Proposition is in Curry logic (three-valued: True, False, Unknown)
    Proposition =.. [LogicalOp | Args],
    (
        LogicalOp = and -> all(member(A, Args), Proposition = true)
        ; LogicalOp = or -> exists(member(A, Args), Proposition = true)
        ; LogicalOp = not -> (Proposition = true <-> innermost_prop(Args) = false)
        ; LogicalOp = implies -> (Proposition = true <-> (\+ antecedent(Args) ; consequent(Args) = true))
        ; LogicalOp = iff -> (Proposition = true <-> (left_side(Args) = right_side(Args)))
    ),
    compute_truth_value(Proposition, PreState, TruthValue),
    (TruthValue = true ; TruthValue = unknown).  % Permit if true or unknown (safe default)

compute_truth_value(Proposition, State, TruthValue) :-
    Proposition =.. [Pred | Args],
    (
        call(Pred, State, Args)
        -> TruthValue = true
        ; (
            \+ negation_as_failure_definite(Pred, Args, State)
            -> TruthValue = unknown
            ;  TruthValue = false
        )
    ).
```

### 6.5 Ω-Verification Integration

Contracts integrate with Ω verification for formal proof checking.

```prolog
omega_verify_contract(Operation, PreState, PostState) :-
    % Generate proof obligations from contract
    contract_for(Operation, Contract),
    generate_proof_obligations(Contract, PreState, PostState, Obligations),
    % Attempt to discharge each obligation
    forall(
        member(Obligation, Obligations),
        (
            (
                omega_discharge(Obligation, Proof)
                -> record_omega_proof(Operation, Obligation, Proof)
                ;  (
                    omega_ask_human(Obligation, Proof),
                    record_omega_proof(Operation, Obligation, Proof)
                )
            )
        )
    ).

generate_proof_obligations(Contract, PreState, PostState, Obligations) :-
    Contract = contract(_, PreConds, PostConds, Invs, _, _),
    % Obligation: postconditions follow from preconditions
    Obligations = [
        obligation(postcond_follows_from_precond(PreConds, PostConds, PreState, PostState)),
        obligation(invariants_preserved(Invs, PreState, PostState)),
        obligation(no_undefined_behavior(PreState, PostState)),
        obligation(execution_deterministic_if_required(PreState, PostState))
    ].

omega_discharge(Obligation, Proof) :-
    % Try automated theorem proving
    (
        smt_solve(Obligation, Proof)
        ; lean4_prove(Obligation, Proof)
        ; coq_prove(Obligation, Proof)
    ).

omega_ask_human(Obligation, Proof) :-
    % Interactive proof: present to user via Ω interface
    present_obligation_to_user(Obligation),
    accept_proof_from_user(Proof).
```

---

## 7. Failure Modes and Rejection Criteria

### 7.1 Precondition Violations

```prolog
precondition_violation(Operation, State, Violation) :-
    contract_for(Operation, Contract),
    Contract = contract(_, PreConds, _, _, _, _),
    member(PreCond, PreConds),
    \+ precondition_holds(PreCond, State),
    Violation = violation(precondition, PreCond, Operation, why_failed(PreCond, State)).

rejection_reason(precondition_violation(PreCond, State)) :-
    format_rejection(
        'Precondition ~w failed: ~w',
        [PreCond, explain_failure(PreCond, State)]
    ).

explain_failure(type_check(Var, Type), State) :-
    state_binding(State, Var, Value),
    actual_type(Value, ActualType),
    format('Variable ~w has type ~w, expected ~w', [Var, ActualType, Type]).

explain_failure(state_validity(Condition), State) :-
    format('State condition ~w does not hold', [Condition]).

explain_failure(null_check(Var), State) :-
    state_binding(State, Var, Value),
    (Value = null
        -> format('Variable ~w is null', [Var])
        ;  format('Variable ~w is undefined', [Var])
    ).

explain_failure(dependency_check(Dep), State) :-
    \+ state_binding(State, Dep, _),
    format('Required dependency ~w is not available', [Dep]).
```

### 7.2 Postcondition Violations

```prolog
postcondition_violation(Operation, PreState, PostState, Violation) :-
    contract_for(Operation, Contract),
    Contract = contract(_, _, PostConds, _, _, _),
    member(PostCond, PostConds),
    \+ postcondition_holds(PostCond, PreState, PostState),
    Violation = violation(postcondition, PostCond, Operation, 
                         why_postcond_failed(PostCond, PreState, PostState)).

rejection_reason(postcondition_violation(PostCond, PreState, PostState)) :-
    format_rejection(
        'Postcondition ~w failed after operation',
        [PostCond]
    ),
    format('  Pre-state: ~w~n', [summarize_state(PreState)]),
    format('  Post-state: ~w~n', [summarize_state(PostState)]),
    format('  Reason: ~w~n', [explain_postfailure(PostCond, PreState, PostState)]).

explain_postfailure(result_valid(Type), PreState, PostState) :-
    state_binding(PostState, result, Result),
    actual_type(Result, ActualType),
    format('Result has type ~w, expected ~w', [ActualType, Type]).

explain_postfailure(invariant_preserved(Inv), PreState, PostState) :-
    evaluate_invariant(Inv, PreState),
    \+ evaluate_invariant(Inv, PostState),
    format('Invariant ~w held before but not after', [Inv]).
```

### 7.3 Invariant Violations

```prolog
invariant_violation(Operation, PreState, PostState, Violation) :-
    contract_for(Operation, Contract),
    Contract = contract(_, _, _, Invs, _, _),
    member(Invariant, Invs),
    Invariant = inv(Name, Pred, Scope),
    (
        (Scope = class, \+ evaluate_invariant(Pred, PreState))
        ; (Scope = class, \+ evaluate_invariant(Pred, PostState))
        ; (Scope = global, \+ invariant_holds_globally(Pred, PostState))
    ),
    Violation = violation(invariant, Invariant, Operation,
                         why_invariant_violated(Name, Pred, PreState, PostState)).

rejection_reason(invariant_violation(Name, Pred, PreState, PostState)) :-
    (
        evaluate_invariant(Pred, PreState),
        \+ evaluate_invariant(Pred, PostState)
        -> format_rejection(
            'Invariant ~w violated: held before but not after',
            [Name]
        )
        ;  format_rejection(
            'Invariant ~w violated: does not hold in result state',
            [Name]
        )
    ),
    format('  Invariant: ~w~n', [Pred]),
    format('  Pre-state satisfies: ~w~n', [evaluate_invariant(Pred, PreState)]),
    format('  Post-state satisfies: ~w~n', [evaluate_invariant(Pred, PostState)]).
```

### 7.4 Type Violations

```prolog
type_violation(Operation, PreState, PostState, Violation) :-
    (
        % Type mismatch in arguments
        member(binding(Var, Type), postcondition_bindings(Operation)),
        state_binding(PostState, Var, Value),
        \+ type_of(Value, Type)
        -> Violation = violation(type_mismatch, binding(Var, Type, actual(value_type(Value))), 
                                Operation, why_type_mismatch)
        ; % Type error in intermediate computation
        execution_trace(Operation, PreState, Trace),
        member(Step, Trace),
        type_error_in_step(Step, State),
        Violation = violation(type_error, Step, Operation, why_type_error)
    ).

type_error_in_step(Step, State) :-
    (
        Step = assignment(Var, Expr),
        declared_type(Var, Type),
        \+ type_of(Expr, Type)
    ; (
        Step = method_call(Recv, Method, Args),
        class_of(Recv, Class),
        \+ method_exists_with_signature(Class, Method, Args)
    ) ; (
        Step = array_access(Arr, Index),
        array_type(Arr, ElementType),
        \+ index_type_valid(Index)
    )
    ).

rejection_reason(type_violation(binding(Var, ExpectedType, actual(ActualType)), _)) :-
    format_rejection(
        'Type mismatch for variable ~w~n  Expected: ~w~n  Actual: ~w',
        [Var, ExpectedType, ActualType]
    ).
```

### 7.5 State Validity Violations

```prolog
state_validity_violation(Operation, PostState, Violation) :-
    (
        % Null pointer violation
        state_binding(PostState, Var, null),
        required_non_null(Var, PostState)
        -> Violation = violation(null_ptr, Var, Operation, 'null pointer')
        ; % Memory corruption
        (
            memory_corruption_detected(PostState),
            Violation = violation(memory_error, unknown, Operation, 'memory corruption')
        )
        ; % Unreachable binding
        (
            state_binding(PostState, Var, _),
            \+ reachable_in_state(Var, PostState),
            Violation = violation(unreachable, Var, Operation, 'binding unreachable')
        )
    ).

rejection_reason(state_validity_violation(null_ptr, Var, _)) :-
    format_rejection(
        'Null pointer violation: required variable ~w is null',
        [Var]
    ).

rejection_reason(state_validity_violation(memory_error, _, _)) :-
    format_rejection('Memory corruption detected in post-state', []).

rejection_reason(state_validity_violation(unreachable, Var, _)) :-
    format_rejection(
        'Binding unreachable: variable ~w is not reachable from heap roots',
        [Var]
    ).
```

### 7.6 Concurrency Violations

```prolog
concurrency_violation(Operation, PreState, PostState, Violation) :-
    (
        % Race condition detected
        concurrent_write_detected(Operation, PreState, PostState)
        -> Violation = violation(race_condition, Operation, Operation, 'concurrent write')
        ; % Deadlock detected
        (
            deadlock_detected(Operation, PostState),
            Violation = violation(deadlock, Operation, Operation, 'circular lock wait')
        )
        ; % Lock held violation
        (
            required_lock_not_held(Operation, PreState),
            Violation = violation(lock_required, Operation, Operation, 'lock not held')
        )
    ).

rejection_reason(concurrency_violation(race_condition, Op, _)) :-
    format_rejection(
        'Race condition: operation ~w conflicts with concurrent operations',
        [Op]
    ).

rejection_reason(concurrency_violation(deadlock, Op, _)) :-
    format_rejection(
        'Deadlock detected: operation ~w would create circular lock wait',
        [Op]
    ).
```

### 7.7 Rejection Protocol

```prolog
reject_transition(Reason, Operation, PreState, PostState) :-
    % 1. Classify violation
    violation_class(Reason, ViolationClass),
    % 2. Generate explanation
    generate_rejection_explanation(Reason, Operation, PreState, PostState, Explanation),
    % 3. Propose repair suggestions
    suggest_repairs(Reason, Operation, PreState, Suggestions),
    % 4. Record rejection for debugging
    record_rejection(ViolationClass, Reason, Operation, Explanation, Suggestions),
    % 5. Report to caller
    report_rejection(ViolationClass, Reason, Explanation, Suggestions),
    % 6. Fail the transition
    !, fail.

violation_class(Reason, ViolationClass) :-
    Reason = violation(Type, _, _, _),
    (
        Type = precondition -> ViolationClass = precondition_failure
        ; Type = postcondition -> ViolationClass = postcondition_failure
        ; Type = invariant -> ViolationClass = invariant_violation
        ; Type = type_mismatch -> ViolationClass = type_error
        ; Type = type_error -> ViolationClass = type_error
        ; Type = null_ptr -> ViolationClass = runtime_error
        ; Type = memory_error -> ViolationClass = runtime_error
        ; Type = race_condition -> ViolationClass = concurrency_error
        ; Type = deadlock -> ViolationClass = concurrency_error
        ; Type = _ -> ViolationClass = unknown_error
    ).

generate_rejection_explanation(Reason, Operation, PreState, PostState, Explanation) :-
    format(atom(Explanation),
        'Operation ~w was rejected due to ~w~n~nPre-state: ~w~nPost-state: ~w~n~nFull reason: ~w',
        [Operation, reason_short(Reason), summarize_state(PreState), 
         summarize_state(PostState), Reason]).

suggest_repairs(violation(precondition, PreCond, _, _), Operation, PreState, Suggestions) :-
    precondition_repair_suggestions(PreCond, Operation, PreState, Suggestions).

suggest_repairs(violation(invariant, inv(Name, _, _), _, _), Operation, PreState, Suggestions) :-
    format(atom(S1),
        'Ensure all class invariants are satisfied before calling ~w',
        [Operation]),
    format(atom(S2),
        'Check that the pre-state satisfies invariant: ~w',
        [Name]),
    Suggestions = [S1, S2].

precondition_repair_suggestions(null_check(Var), Operation, State, Suggestions) :-
    format(atom(S1),
        'Initialize ~w before calling ~w',
        [Var, Operation]),
    Suggestions = [S1].

precondition_repair_suggestions(type_check(Var, ExpectedType), Operation, State, Suggestions) :-
    state_binding(State, Var, Value),
    actual_type(Value, ActualType),
    format(atom(S1),
        'Convert ~w from ~w to ~w before calling ~w',
        [Var, ActualType, ExpectedType, Operation]),
    Suggestions = [S1].

report_rejection(ViolationClass, Reason, Explanation, Suggestions) :-
    format('REJECTION: ~w~n', [ViolationClass]),
    format('~w~n', [Explanation]),
    forall(member(S, Suggestions), format('  - ~w~n', [S])).
```

---

## 8. Integration with Curry Logic

### 8.1 Three-Valued Logic

Contracts support Curry logic's three truth values: **True**, **False**, **Unknown**.

```prolog
curry_truth_value(Proposition, State, Value) :-
    (
        % Provably true
        provably_true(Proposition, State)
        -> Value = true
        ; % Provably false
        (
            provably_false(Proposition, State)
            -> Value = false
            ; % Unknown (undefined or non-deterministic)
            Value = unknown
        )
    ).

provably_true(Proposition, State) :-
    % Standard Prolog: succeeds
    call(Proposition, State).

provably_false(Proposition, State) :-
    % Explicit negation: provably negated
    \+ call(Proposition, State),
    negation_is_definite(Proposition, State).

negation_is_definite(Proposition, State) :-
    % Negation is "definite" if either:
    % 1. The proposition is ground (all variables bound)
    is_ground(Proposition),
    % 2. Or we have explicit negative information
    \+ unknown_predicate(Proposition).

unknown_predicate(Proposition) :-
    functor(Proposition, Pred, Arity),
    \+ (
        predicate_defined(Pred, Arity)
        ; builtin_predicate(Pred, Arity)
    ).
```

### 8.2 Curry Preconditions

Preconditions can require propositions to be **true or unknown** (safe default).

```prolog
curry_precondition_satisfied(PreCond, State) :-
    curry_truth_value(PreCond, State, Value),
    (
        Value = true   % Definitely true
        ; Value = unknown  % Unknown: conservative (don't block)
    ).

curry_postcondition_required(PostCond, PreState, PostState) :-
    curry_truth_value(PostCond, PostState, Value),
    (
        Value = true   % Must be provably true
    ),
    \+ (
        curry_truth_value(PostCond, PreState, PreValue),
        PreValue = true,
        curry_truth_value(PostCond, PostState, false)
    ).
```

### 8.3 Partial Function Contracts

Contracts express partial function semantics.

```prolog
partial_function_contract(Function, Arguments) :-
    % Precondition: arguments in domain
    domain_membership(Function, Arguments),
    % Postcondition: result is defined and correct
    (
        apply_partial(Function, Arguments, Result)
        -> result_is_defined(Function, Arguments, Result)
        ;  function_undefined_on_args(Function, Arguments)
    ).

domain_membership(Function, Args) :-
    curry_truth_value(in_domain(Function, Args), context, Value),
    (Value = true ; Value = unknown).  % Safe default

apply_partial(Function, Arguments, Result) :-
    function_definition(Function, Body),
    safe_call(Body, Arguments, Result).

safe_call(Body, Arguments, Result) :-
    catch(
        call(Body, Arguments, Result),
        _Error,
        fail
    ).
```

---

## 9. Integration with Ω Verification

### 9.1 Ω Verification Points

Contracts are verified at five points in the Ω verification process.

```prolog
omega_verify_contract(Operation, PreState, PostState, Level) :-
    % Level 1: Syntax verification
    verify_contract_syntactically(Operation),
    % Level 2: Type checking
    verify_contract_types(Operation, PreState),
    % Level 3: Precondition discharge
    discharge_preconditions(Operation, PreState),
    % Level 4: Execution correctness
    verify_execution_correct(Operation, PreState, PostState),
    % Level 5: Postcondition and invariant verification
    verify_postconditions_and_invariants(Operation, PreState, PostState),
    Level = full_verification.
```

### 9.2 Proof Obligation Generation

Each contract generates proof obligations for the Ω prover.

```prolog
generate_omega_obligations(Contract, PreState, PostState, Obligations) :-
    Contract = contract(ID, PreConds, PostConds, Invs, AdmRule, _),
    % Obligation 1: Preconditions are satisfiable
    Obligations = [
        obligation(1, precon_satisfiable, forall_true(PreConds, PreState)),
        % Obligation 2: Postconditions follow from preconditions and operation
        obligation(2, postcon_follows, implications(
            conj(PreConds),
            conj(PostConds)
        )),
        % Obligation 3: Invariants preserved
        obligation(3, inv_preserved, forall(
            member(Inv, Invs),
            (evaluate_invariant(Inv, PreState), evaluate_invariant(Inv, PostState))
        )),
        % Obligation 4: No undefined behavior
        obligation(4, no_undef_behavior, \+ undefined_behavior(Operation, PreState, PostState)),
        % Obligation 5: Admissibility (admissibility rule callable)
        obligation(5, admissible, call(AdmRule, PreState, Operation, PostState))
    ].
```

### 9.3 Ω Proof Format

Proofs are expressed in Ω format for cryptographic commitment.

```prolog
omega_proof_format(Proof) :-
    Proof = omega_proof(
        contract_id(ContractID),
        pre_state(PreStateHash),
        post_state(PostStateHash),
        obligations_discharged(ObligationProofs),
        timestamp(Timestamp),
        signature(EdSignature),
        certificate(PublicKey)
    ).

generate_omega_proof(Contract, PreState, PostState, Proof) :-
    Contract = contract(ContractID, _, _, _, _, _),
    blake3_hash(PreState, PreStateHash),
    blake3_hash(PostState, PostStateHash),
    generate_omega_obligations(Contract, PreState, PostState, Obligations),
    maplist(discharge_obligation, Obligations, ObligationProofs),
    get_timestamp(Timestamp),
    ed25519_sign_proof(
        omega_proof(
            contract_id(ContractID),
            pre_state(PreStateHash),
            post_state(PostStateHash),
            obligations_discharged(ObligationProofs),
            timestamp(Timestamp)
        ),
        PrivateKey,
        EdSignature
    ),
    get_public_key(PrivateKey, PublicKey),
    Proof = omega_proof(
        contract_id(ContractID),
        pre_state(PreStateHash),
        post_state(PostStateHash),
        obligations_discharged(ObligationProofs),
        timestamp(Timestamp),
        signature(EdSignature),
        certificate(PublicKey)
    ).
```

### 9.4 Ω Certification

Contracts are certified by the Ω verification engine.

```prolog
omega_certify_contract(Proof, CertificationResult) :-
    Proof = omega_proof(_, PreStateHash, PostStateHash, ObligationProofs, _, EdSignature, PublicKey),
    % Verify signature
    verify_signature(EdSignature, PublicKey, Proof),
    % Verify all obligations discharged
    forall(
        member(obligation_proof(ObligationID, _), ObligationProofs),
        obligation_valid(ObligationID, ObligationProofs)
    ),
    % Generate certification
    CertificationResult = certification(
        status(verified),
        proof(Proof),
        timestamp(timestamp_now),
        verifier(omega_engine)
    ).
```

---

## 10. Complete Contract Examples

### 10.1 Queue.enqueue Full Example

```prolog
% Full contract specification for Queue.enqueue(x: Integer)

contract_specification(
    queue_enqueue,
    {
        id: 'queue_enqueue_v1',
        operation: method(queue, enqueue, [integer]),
        
        preconditions: [
            % Type precondition
            type_correct(receiver, queue),
            type_correct(arg(1), integer),
            
            % State preconditions
            queue_initialized(receiver),
            \+ queue_locked(receiver),
            
            % Dependency preconditions
            queue_capacity_available(receiver),
            
            % Invariant preconditions
            queue_valid_before(receiver),
            
            % Access control
            method_accessible(enqueue, caller)
        ],
        
        postconditions: [
            % State validity
            queue_valid_after(receiver),
            
            % Invariant preservation
            fifo_order_preserved(receiver),
            enqueued_element_at_tail(receiver, arg(1)),
            
            % Effect recording
            effect_recorded(enqueue, receiver, arg(1), timestamp_now),
            
            % Output validity
            no_return_value(enqueue),
            
            % Ordering
            enqueue_completes_atomically(receiver)
        ],
        
        invariants: [
            inv(queue_structure, is_list(queue_contents), class),
            inv(fifo_property, all_dequeued_in_order, global),
            inv(element_presence, element_in_iff_in_history, class)
        ],
        
        admissibility: queue_enqueue_admissible,
        
        evidence: proof_commitment
    }
).

% Admissibility rule
queue_enqueue_admissible(Queue, Element, PreState, PostState) :-
    % Receiver valid
    \+ null_or_undefined(Queue),
    is_queue(Queue),
    
    % Element valid
    integer(Element),
    
    % Pre-state consistency
    queue_valid_pre(Queue, PreState),
    
    % No capacity violation
    queue_has_space(Queue, PreState),
    
    % Post-state correctness
    queue_contents_after(Queue, PreState, PostState, Contents),
    last(Contents, Element),
    length(Contents, Len),
    length(queue_contents_before(Queue, PreState), LenBefore),
    Len is LenBefore + 1.

% Verification
verify_queue_enqueue(Queue, Element, PreState, PostState) :-
    verify_and_commit(
        method(queue, enqueue, [Element]),
        PreState,
        PostState
    ).
```

### 10.2 BinarySearchTree.insert Full Example

```prolog
% Full contract specification for BST.insert(x: Integer)

contract_specification(
    bst_insert,
    {
        id: 'bst_insert_v1',
        operation: method(bst, insert, [integer]),
        
        preconditions: [
            type_correct(receiver, bst),
            type_correct(arg(1), integer),
            bst_initialized(receiver),
            \+ bst_locked(receiver),
            \+ bst_full(receiver),
            bst_valid_before(receiver),
            method_accessible(insert, caller)
        ],
        
        postconditions: [
            bst_valid_after(receiver),
            bst_invariant_preserved(receiver),
            inserted_element_in_tree(receiver, arg(1)),
            bst_search_correct_after(receiver),
            tree_height_bounded(receiver)
        ],
        
        invariants: [
            inv(bst_property, forall_node_bst_satisfied, class),
            inv(element_presence, element_in_iff_in_history, class),
            inv(no_duplicates, all_elements_unique, class),
            inv(tree_structure, tree_is_acyclic, global)
        ],
        
        admissibility: bst_insert_admissible,
        
        evidence: proof_commitment
    }
).

% Verification
bst_insert_admissible(Tree, Element, PreState, PostState) :-
    % Receiver valid
    \+ null_or_undefined(Tree),
    is_bst(Tree),
    
    % Element valid and new
    integer(Element),
    \+ bst_contains(Tree, Element, PreState),
    
    % Pre-state: valid BST
    bst_valid_pre(Tree, PreState),
    
    % Post-state: valid BST with element inserted
    bst_contains(Tree, Element, PostState),
    bst_valid_post(Tree, PostState),
    tree_height_valid(Tree, PostState),
    \+ tree_unbalanced(Tree, PostState).
```

### 10.3 HashMap.put Full Example

```prolog
contract_specification(
    hashmap_put,
    {
        id: 'hashmap_put_v1',
        operation: method(hashmap, put, [key, value]),
        
        preconditions: [
            type_correct(receiver, hashmap),
            type_correct(arg(1), key),
            type_correct(arg(2), value),
            hashmap_initialized(receiver),
            \+ hashmap_locked(receiver),
            load_factor_acceptable(receiver),
            hashmap_valid_before(receiver),
            method_accessible(put, caller)
        ],
        
        postconditions: [
            hashmap_valid_after(receiver),
            hashmap_invariants_preserved(receiver),
            key_value_pair_stored(receiver, arg(1), arg(2)),
            lookup_returns_stored_value(receiver, arg(1), arg(2)),
            all_other_bindings_preserved(receiver, arg(1))
        ],
        
        invariants: [
            inv(hash_consistency, hash_table_consistent, class),
            inv(collision_handling, collision_resolution_valid, class),
            inv(load_factor, load_factor_maintained, class)
        ],
        
        admissibility: hashmap_put_admissible,
        
        evidence: proof_commitment
    }
).

hashmap_put_admissible(Map, Key, Value, PreState, PostState) :-
    % Receiver valid
    \+ null_or_undefined(Map),
    is_hashmap(Map),
    
    % Arguments valid
    valid_key(Key),
    valid_value(Value),
    
    % Pre-state valid
    hashmap_valid_pre(Map, PreState),
    
    % Post-state: key-value pair stored and retrievable
    hashmap_get(Map, Key, PostState, StoredValue),
    StoredValue = Value,
    
    % All other entries preserved (except potentially overwritten key)
    forall(
        hashmap_entry(Map, K, V, PreState),
        (
            K \= Key
            -> hashmap_get(Map, K, PostState, V)
            ;  true  % Key may have been overwritten
        )
    ),
    
    % Load factor acceptable
    load_factor_acceptable(Map, PostState).
```

---

## 11. Formal Prolog Library

Complete Prolog predicates for contract definition, verification, and enforcement.

```prolog
% ============================================================================
% Contract Definition Predicates
% ============================================================================

% Define a new contract
define_contract(ID, Preconditions, Postconditions, Invariants, 
                Admissibility, Evidence) :-
    is_valid_uuid(ID),
    is_list(Preconditions),
    is_list(Postconditions),
    is_list(Invariants),
    callable(Admissibility),
    callable(Evidence),
    assertz(contract(ID, Preconditions, Postconditions, Invariants, 
                    Admissibility, Evidence)).

% ============================================================================
% Precondition Predicates
% ============================================================================

% Verify all preconditions
verify_all_preconditions(Operation, State) :-
    contract_for(Operation, Contract),
    Contract = contract(_, PreConds, _, _, _, _),
    forall(
        member(PreCond, PreConds),
        precondition_holds(PreCond, State)
    ).

% Individual precondition evaluation
precondition_holds(type_check(Var, Type), State) :-
    state_binding(State, Var, Value),
    type_of(Value, Type).

precondition_holds(non_null(Var), State) :-
    state_binding(State, Var, Value),
    Value \= null.

precondition_holds(state_valid(Predicate), State) :-
    call(Predicate, State).

precondition_holds(dependency(Dep), State) :-
    state_binding(State, Dep, _),
    binding_initialized(Dep, State).

% ============================================================================
% Postcondition Predicates
% ============================================================================

% Verify all postconditions
verify_all_postconditions(Operation, PreState, PostState) :-
    contract_for(Operation, Contract),
    Contract = contract(_, _, PostConds, _, _, _),
    forall(
        member(PostCond, PostConds),
        postcondition_holds(PostCond, PreState, PostState)
    ).

% Individual postcondition evaluation
postcondition_holds(result_valid(Type), _, PostState) :-
    state_binding(PostState, result, Result),
    type_of(Result, Type).

postcondition_holds(invariant_preserved(Inv), PreState, PostState) :-
    evaluate_invariant(Inv, PreState),
    evaluate_invariant(Inv, PostState).

postcondition_holds(effect_recorded(Effect), _, PostState) :-
    state_binding(PostState, effect_log, Log),
    member(Effect, Log).

% ============================================================================
% Invariant Predicates
% ============================================================================

% Verify all invariants hold in state
verify_all_invariants(State) :-
    forall(
        class_invariant(_, Invariant),
        evaluate_invariant(Invariant, State)
    ).

% Evaluate an invariant in a state
evaluate_invariant(inv(_, Predicate, _), State) :-
    Predicate =.. [Pred | Args],
    call(Pred, State, Args).

% Verify invariant preserved across transition
invariant_preserved(inv(Name, Pred, Scope), PreState, PostState) :-
    (
        Scope = class
        -> (
            evaluate_invariant(inv(Name, Pred, Scope), PreState),
            evaluate_invariant(inv(Name, Pred, Scope), PostState)
        )
        ; Scope = global
        -> evaluate_invariant(inv(Name, Pred, Scope), PostState)
        ; fail
    ).

% ============================================================================
% Admissibility Predicates
% ============================================================================

% Check if transition is admissible
is_admissible(Operation, PreState, PostState) :-
    contract_for(Operation, Contract),
    Contract = contract(_, PreConds, PostConds, Invs, AdmRule, _),
    verify_all_preconditions(Operation, PreState),
    verify_all_postconditions(Operation, PreState, PostState),
    forall(member(Inv, Invs), invariant_preserved(Inv, PreState, PostState)),
    call(AdmRule, PreState, Operation, PostState),
    \+ violation_exists(PreState, PostState, Operation).

% ============================================================================
% Verification and Commitment
% ============================================================================

% Verify and commit a state transition
verify_and_commit(Operation, PreState, PostState) :-
    % Phase 1: Contract structure verification
    verify_contract_structure(Operation),
    % Phase 2: Precondition verification
    verify_all_preconditions(Operation, PreState),
    % Phase 3: Execution verification
    verify_execution_sound(Operation, PreState, PostState),
    % Phase 4: Postcondition verification
    verify_all_postconditions(Operation, PreState, PostState),
    % Phase 5: Invariant verification
    forall(
        class_invariant(_, Inv),
        invariant_preserved(Inv, PreState, PostState)
    ),
    % Phase 6: Admissibility verification
    is_admissible(Operation, PreState, PostState),
    % Phase 7: Proof generation
    generate_proof_commitment(Operation, PreState, PostState, Proof),
    % Phase 8: Atomic commit
    commit_atomic(PostState, Proof).

% ============================================================================
% Failure and Rejection
% ============================================================================

% Reject transition with reason
reject_transition(Reason, Operation, PreState, PostState) :-
    generate_rejection_explanation(Reason, Operation, PreState, PostState, Explanation),
    suggest_repairs(Reason, Operation, PreState, Suggestions),
    record_rejection(Reason, Operation, Explanation, Suggestions),
    report_rejection(Reason, Explanation, Suggestions),
    !, fail.

% ============================================================================
% Utility Predicates
% ============================================================================

% Get contract for an operation
contract_for(Operation, Contract) :-
    operation_id(Operation, OperationID),
    contract(OperationID, _, _, _, _, _),
    Contract = contract(OperationID, _, _, _, _, _).

% Check if state is valid
is_state_valid(State) :-
    \+ null_or_undefined(State),
    state_structure_valid(State),
    verify_all_invariants(State),
    \+ memory_corruption_exists(State).

% Create new contract ID
new_contract_id(UUID) :-
    generate_uuid(UUID).

% Record verification result
record_verification(Category, Item, Result) :-
    get_timestamp(Timestamp),
    assertz(verification_record(Category, Item, Result, Timestamp)).

% Generate UUID
generate_uuid(UUID) :-
    uuid_library:uuid(UUID, [random]).

% Get current timestamp
get_timestamp(Timestamp) :-
    get_time(UnixTime),
    Timestamp is UnixTime.
```

---

## 12. Implementation Checklist

- [ ] **Contract Definition**: All contract types registered
- [ ] **Precondition Engine**: Type, state, dependency, invariant, access control checks
- [ ] **Postcondition Engine**: State validity, invariant preservation, effect recording
- [ ] **Admissibility Checker**: All violation categories detected
- [ ] **Verification Pipeline**: 8-phase verification process
- [ ] **Curry Logic Integration**: Three-valued logic support
- [ ] **Ω Integration**: Proof obligation generation and verification
- [ ] **Rejection Handler**: Complete rejection protocol with suggestions
- [ ] **WORM Commitment**: Atomic state transitions with proofs
- [ ] **Cryptographic Signing**: Ed25519 proof signatures
- [ ] **Test Suite**: Contract satisfaction tests for all 5 contract types
- [ ] **Documentation**: Integration guide for OMEGA-MUSTACHE system

---

## 13. References and Integration Points

1. **OMEGA-MUSTACHE Core**: State machine, execution engine, effect logging
2. **Curry Logic Module**: Three-valued propositions, partial function semantics
3. **Ω Verification Engine**: Proof obligation discharge, SMT solving, Lean4/Coq backends
4. **WORM Log**: Immutable proof commitment storage, cryptographic sealing
5. **Concurrency Layer**: Lock management, race condition detection
6. **Type System**: Type inference, subtyping, union types
7. **Memory Model**: Heap management, reachability analysis, garbage collection

---

**Status**: Complete formal specification, ready for implementation.  
**Next Step**: Integrate with execution engine and wire contract verification into state transition pipeline.
