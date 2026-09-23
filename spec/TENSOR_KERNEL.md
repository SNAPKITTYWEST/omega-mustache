<!-- SPDX-License-Identifier: AGPL-3.0-or-later OR Apache-2.0 -->
<!-- CLONE_GATE:AES256:50e0313ba329c7921e3bb9532d8ba9ca879161138a69a80db36eef3776482f68 -->

# K Tensor Array Kernel Specification

## 1. Core Tensor Definition

```prolog
% A tensor is an N-dimensional array with typed elements

tensor(A) :-
    tensor_id(A, ID),
    element_type(A, Type),
    rank(A, N),
    shape(A, [D1, D2, ..., Dn]),
    strides(A, [S1, S2, ..., Sn]),
    values(A, Values),
    canonical_hash(A, Hash).

% Properties:
% - ID: Globally unique identifier
% - Type: {int, float, bool, complex, symbol}
% - Rank: N ≥ 0 (0 = scalar, 1 = vector, 2 = matrix, etc.)
% - Shape: Dimensions [D1, D2, ..., Dn], each Di > 0
% - Strides: Memory layout [S1, S2, ..., Sn]
% - Values: Linearized array data
% - Hash: Content-addressed digest
```

### Constraints

```prolog
% (C1) Shape validity
valid_shape(A) :-
    shape(A, Shape),
    forall(member(D, Shape), (integer(D), D > 0)).

% (C2) Element type consistency
consistent_element_type(A) :-
    element_type(A, Type),
    values(A, Values),
    forall(member(V, Values), valid_element(V, Type)).

% (C3) Index domain correctness
valid_index_domain(A) :-
    shape(A, Shape),
    rank(A, N),
    length(Shape, N),
    Index = [I1, I2, ..., In],
    forall(
        (between(1, N, J), nth1(J, Index, Ij), nth1(J, Shape, Dj)),
        (0 =< Ij, Ij < Dj)
    ).

% (C4) Stride consistency
valid_strides(A) :-
    shape(A, Shape),
    strides(A, Strides),
    length(Shape, N),
    length(Strides, N),
    strides_match_shape(Shape, Strides).

% (C5) Closure property (tensor composition is a tensor)
closure_preserved(A ☉ B) :-
    tensor(A),
    tensor(B),
    compatible_shape(A, B),
    Result is element_wise_product(A, B),
    tensor(Result).
```

## 2. Tensor Operations

### ☉ Product (Element-wise)

```prolog
product(A, B, Result) :-
    shape(A, Shape),
    shape(B, Shape),  % Shapes must match
    element_type(A, Type),
    element_type(B, Type),
    findall(
        (V1 * V2),
        (nth0(I, values(A), V1),
         nth0(I, values(B), V2)),
        ResultValues
    ),
    tensor(Result, element_type=Type, shape=Shape, values=ResultValues).

% A ☉ B = [a_i * b_i for i in domain]
```

### ⌹ Partition (Decompose)

```prolog
partition(A, K, Partitions) :-
    rank(A, N),
    K >= 1,
    K =< N,
    % Decompose along first K dimensions
    shape(A, [D1, D2, ..., Dk | Rest]),
    TotalInPartition is product([D1, D2, ..., Dk]),
    NumPartitions is product(Rest),
    % Split values into NumPartitions chunks
    split_values_into_chunks(values(A), TotalInPartition, NumPartitions, Partitions).

% ⌹k(A) = [A_0, A_1, ..., A_{p-1}]
% where p = product(shape(A)[k:])
```

### ○ Closure (Reconstruct)

```prolog
closure(Partitions, Original) :-
    length(Partitions, NumPartitions),
    % Verify all partitions have same shape
    maplist(compatible_shape, Partitions),
    % Concatenate values
    append_all_values(Partitions, ReconstructedValues),
    % Infer shape from partition structure
    InferredShape = infer_closure_shape(Partitions),
    tensor(Original, values=ReconstructedValues, shape=InferredShape).

% ○([A_0, A_1, ..., A_{p-1}]) = A (if partitions came from ⌹k)
```

**Closure Theorem**:
```
○(⌹k(A)) = A
```

Proof: Partition followed by reconstruction with identical k produces original tensor.

### △ Difference (Element-wise Subtraction)

```prolog
difference(A, B, Result) :-
    shape(A, Shape),
    shape(B, Shape),
    element_type(A, Type),
    element_type(B, Type),
    findall(
        (V1 - V2),
        (nth0(I, values(A), V1),
         nth0(I, values(B), V2)),
        ResultValues
    ),
    tensor(Result, element_type=Type, shape=Shape, values=ResultValues).

% A △ B = [a_i - b_i for i in domain]
```

### ◇ Transform (Apply function)

```prolog
transform(A, Func, Result) :-
    element_type(A, Type),
    shape(A, Shape),
    maplist(
        call(Func),
        values(A),
        ResultValues
    ),
    tensor(Result, element_type=output_type(Func), shape=Shape, values=ResultValues).

% ◇f(A) = [f(a_i) for i in domain]
```

### ⬡ Composition (Combined operations)

```prolog
composition(A, B, Func, Result) :-
    tensor(A),
    tensor(B),
    shape(A, Shape),
    shape(B, Shape),
    findall(
        ResultVal,
        (nth0(I, values(A), Va),
         nth0(I, values(B), Vb),
         call(Func, Va, Vb, ResultVal)),
        ResultValues
    ),
    tensor(Result, element_type=result_type(Func), shape=Shape, values=ResultValues).

% A ⬡ B = [f(a_i, b_i) for i in domain]
```

## 3. Index Operations

```prolog
% Get element at index
element_at(A, Index, Value) :-
    values(A, Values),
    index_to_offset(A, Index, Offset),
    nth0(Offset, Values, Value).

% Compute linear offset from multi-dimensional index
index_to_offset(A, [I1, I2, ..., In], Offset) :-
    strides(A, [S1, S2, ..., Sn]),
    Offset is sum([I_j * S_j for j=1..n]).

% Set element at index
set_element(A_in, Index, Value, A_out) :-
    values(A_in, Values_in),
    index_to_offset(A_in, Index, Offset),
    replace_nth0(Offset, Values_in, Value, Values_out),
    A_out = A_in with values=Values_out.
```

## 4. Broadcasting (Optional)

```prolog
% Broadcast smaller tensor to larger shape
broadcast(A, TargetShape, Result) :-
    shape(A, OldShape),
    can_broadcast(OldShape, TargetShape),
    % Repeat elements according to broadcast rules
    broadcast_values(values(A), OldShape, TargetShape, ResultValues),
    tensor(Result, shape=TargetShape, values=ResultValues, element_type=element_type(A)).

can_broadcast(S1, S2) :-
    % Trailing dimensions match, or S1 has dimension 1
    reverse(S1, R1),
    reverse(S2, R2),
    broadcast_compatible(R1, R2).
```

## 5. Type System

```prolog
% Element types
element_type(scalar) :- member(scalar, [int, float, bool, complex, symbol]).

% Type promotion
type_promote(Type1, Type2, Result) :-
    member([Type1, Type2, Result], [
        [int, int, int],
        [int, float, float],
        [float, float, float],
        [bool, bool, bool],
        [complex, complex, complex]
    ]).

% Type coercion
coerce_to_type(Value, int, CoercedValue) :- integer(Value) -> CoercedValue = Value.
coerce_to_type(Value, float, CoercedValue) :- number(Value) -> CoercedValue is float(Value).
coerce_to_type(Value, bool, true) :- Value \= 0, Value \= false.
coerce_to_type(Value, bool, false) :- Value = 0 ; Value = false.
```

## 6. Canonical Hash

```prolog
canonical_hash(A, Hash) :-
    element_type(A, Type),
    shape(A, Shape),
    values(A, Values),
    % Canonical form: deterministically ordered
    blake3(
        atom_string(Type) ++
        term_string(Shape) ++
        term_string(Values),
        Hash
    ).

% Two tensors are semantically equal iff canonical_hash(A) = canonical_hash(B)
```

## 7. Equivalence and Comparison

```prolog
% Semantic equality (same hash)
semantically_equal(A, B) :-
    canonical_hash(A, Hash),
    canonical_hash(B, Hash).

% Structural equality (same structure, possibly different ID)
structurally_equal(A, B) :-
    element_type(A, Type),
    element_type(B, Type),
    shape(A, Shape),
    shape(B, Shape),
    values(A, Va),
    values(B, Vb),
    Va = Vb.

% Element-wise less-than (requires compatible shapes, numeric types)
element_wise_less(A, B, Result) :-
    shape(A, Shape),
    shape(B, Shape),
    element_type(A, numeric_type),
    element_type(B, numeric_type),
    findall(
        (Va < Vb),
        (nth0(I, values(A), Va),
         nth0(I, values(B), Vb)),
        ResultValues
    ),
    tensor(Result, element_type=bool, shape=Shape, values=ResultValues).
```

## 8. Validation

```prolog
% Comprehensive validation
tensor_valid(A) :-
    % All constraints must hold
    valid_shape(A),
    consistent_element_type(A),
    valid_index_domain(A),
    valid_strides(A),
    canonical_hash(A, _).  % Hash must be computable

% Batch validation
all_tensors_valid(Tensors) :-
    maplist(tensor_valid, Tensors).
```

## 9. Immutability

```prolog
% Tensors are content-addressed
% Modifications produce new tensors

immutable(A) :-
    tensor_id(A, ID),
    canonical_hash(A, Hash),
    % ID and Hash are permanent
    \+ can_change_hash_of(ID).

% To modify, create a new tensor with a new ID
modify(A_old, Operation, A_new) :-
    tensor_id(A_new, NewID),
    NewID \= tensor_id(A_old, _),
    % But semantic relationship is recorded
    parent_tensor(A_new, A_old),
    operation_applied(A_new, Operation).
```

## 10. Serialization

```prolog
% Serialize tensor to canonical text form
serialize(A, Text) :-
    element_type(A, Type),
    shape(A, Shape),
    values(A, Values),
    atomic_list_concat([
        'tensor(',
        Type, ',',
        term_string(Shape), ',',
        term_string(Values),
        ')'
    ], Text).

% Deserialize from text
deserialize(Text, A) :-
    atom_string(Text, String),
    split_string(String, "(,)", "", Parts),
    Parts = [TypeStr, ShapeStr, ValuesStr],
    atom_string(Type, TypeStr),
    read_term_from_atom(ShapeStr, Shape, []),
    read_term_from_atom(ValuesStr, Values, []),
    tensor(A, element_type=Type, shape=Shape, values=Values).
```

## 11. Theorems

### Theorem K1: Closure Idempotence

```
○(⌹k(A)) = A
```

Proof: Partition splits A into k-dimensional chunks. Closure reconstructs in identical order.

### Theorem K2: Product Associativity

```
(A ☉ B) ☉ C = A ☉ (B ☉ C)
```

Proof: Element-wise product is associative on numeric types.

### Theorem K3: Partition Completeness

```
union(⌹k(A)) = A
```

Proof: All elements of A appear in exactly one partition; union reconstructs original.

### Theorem K4: Transform Composition

```
◇(f ∘ g)(A) = ◇f(◇g(A))
```

Proof: Function composition distributes over element-wise application.

### Theorem K5: Type Safety

```
∀ Operation, ∀ A, B (tensor_valid(A) ∧ compatible_types(A, B))
⟹ tensor_valid(Operation(A, B))
```

Proof: All operations preserve tensor validity on type-compatible inputs.

## 12. Performance Characteristics

```
Operation     | Time Complexity | Space Complexity
──────────────┼─────────────────┼──────────────────
element_at    | O(1)            | O(1)
set_element   | O(N)            | O(N) [creates new]
product ☉     | O(N)            | O(N)
partition ⌹k  | O(N)            | O(N)
closure ○     | O(N)            | O(N)
transform ◇   | O(N)            | O(N)
composition ⬡ | O(N)            | O(N)
```

## 13. Adversarial Tests

```prolog
test_zero_rank_tensor :-
    % Scalar: rank 0, shape [], single element
    tensor(T, rank=0, shape=[], values=[42]),
    element_type(T, int),
    canonical_hash(T, Hash),
    write('Zero-rank tensor: PASS'), nl.

test_partition_reconstruction :-
    % Create tensor, partition, reconstruct, verify
    tensor(A, shape=[2,3], values=[1,2,3,4,5,6]),
    partition(A, 1, Parts),
    closure(Parts, B),
    canonical_hash(A, HA),
    canonical_hash(B, HB),
    HA = HB,  % Must be identical
    write('Partition reconstruction: PASS'), nl.

test_type_coercion :-
    % Mixing types in product
    tensor(A, element_type=int, values=[1,2,3]),
    tensor(B, element_type=float, values=[1.5,2.5,3.5]),
    catch(
        (product(A, B, _), fail),
        type_error,
        write('Type error correctly raised: PASS')
    ), nl.

test_stride_validity :-
    % Invalid strides should fail validation
    catch(
        (tensor(T, shape=[2,3], strides=[6,1], values=[1,2,3,4,5,6]),
         tensor_valid(T), fail),
        validation_error,
        write('Invalid strides rejected: PASS')
    ), nl.

test_shape_consistency :-
    % Shape-value mismatch
    catch(
        (shape_mismatch([2,3], [1,2,3,4])),  % 6 values, but 4 provided
        error,
        write('Shape-value mismatch rejected: PASS')
    ), nl.
```

## 14. K-Kernel Integration Points

The K tensor kernel:

1. **Takes input** from Go orchestration (typed)
2. **Executes operations** deterministically (prolog)
3. **Produces tensor states** with canonical hashes
4. **Validates all constraints** before returning
5. **Records lineage** (parent_tensor, operation_applied)
6. **Provides immutable results** (new tensor IDs)
7. **Returns to Go** with verification data

All operations **MUST** succeed in all validation checks or the entire transition is rejected.
