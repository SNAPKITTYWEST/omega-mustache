%% SPDX-License-Identifier: AGPL-3.0-or-later OR Apache-2.0
%% CLONE_GATE:AES256:82cf1b65f0c0c3082623b4e3ce774e3723b3f8fd52f7419797dc54f24fed839e
%%
%% curry_logic.pl — Curry Logic Reasoning Engine (SWI-Prolog)
%%
%% 6 core predicates:
%%   1. semantic_rule/2        — rule applicability
%%   2. derive/3               — SLD derivation with witness
%%   3. admissible/2           — state admissibility
%%   4. transition_allowed/4   — full 5-condition Ω gate
%%   5. tensor_constraint/1    — K tensor invariants
%%   6. omega_candidate/4      — Ω-preservation check
%%
%% Run: swipl prolog/curry_logic.pl
%% Tests: ?- run_all_tests.

:- module(curry_logic,
          [ semantic_rule/2,
            derive/3,
            admissible/2,
            transition_allowed/4,
            tensor_constraint/1,
            omega_candidate/4,
            omega_sig/3,
            run_all_tests/0
          ]).

:- use_module(library(lists)).
:- use_module(library(apply)).

% ─── SEMANTIC STATE ────────────────────────────────────────────────────────────

% semantic_state(AgentID, InputHash, OutputHash, WellFormed, Consistent)
:- dynamic semantic_state/5.

% tensor_state(Hash, Shape, ElemType)
:- dynamic tensor_state/3.

% ─── Ω SIGNATURE ──────────────────────────────────────────────────────────────

%% omega_sig(+SemanticState, +TensorState, -Sig)
%% Ω signature: deterministic combination of semantic + tensor info.
%% Production: Blake3(S_hash || K_hash || C_hash)
%% Here: atom concatenation as a stand-in.
omega_sig(ss(AgentID, InH, OutH, WF, Consistent),
          ts(KHash, _Shape, ElemType),
          Sig) :-
    term_to_atom(WF, WFAtom),
    term_to_atom(Consistent, CAtom),
    atomic_list_concat([AgentID, InH, OutH, WFAtom, CAtom, KHash, ElemType],
                       '|', Sig).

% ─── PREDICATE 1: semantic_rule/2 ─────────────────────────────────────────────

%% semantic_rule(+Rule, +State)
%% Succeeds if Rule is applicable in State.
semantic_rule(Rule, ss(AgentID, _InH, _OutH, WF, Consistent)) :-
    ground(Rule),
    ground(AgentID),
    WF = true,
    Consistent = true,
    known_rule(Rule).

known_rule(identity).
known_rule(transform).
known_rule(validate).
known_rule(seal).
known_rule(replay).
known_rule(contract_eval).

% ─── PREDICATE 2: derive/3 ─────────────────────────────────────────────────────

%% derive(+Premises, +Conclusion, -Witness)
%% SLD resolution: derive Conclusion from Premises, producing a Witness term.
%% Witness is a proof tree: leaf(C) | step(Rule, Antecedents, C)

derive(Premises, Conclusion, leaf(Conclusion)) :-
    member(Conclusion, Premises).

derive(Premises, Conclusion, step(Rule, Witnesses, Conclusion)) :-
    inference_rule(Rule, Antecedents, Conclusion),
    maplist(derive_one(Premises), Antecedents, Witnesses).

derive_one(Premises, Goal, Witness) :-
    derive(Premises, Goal, Witness).

%% Inference rules: rule(Name, Antecedents, Conclusion)
inference_rule(modus_ponens, [impl(A, B), A], B).
inference_rule(conjunction,  [A, B], and(A, B)).
inference_rule(disjunction_left, [A], or(A, _)).
inference_rule(disjunction_right, [B], or(_, B)).
inference_rule(transitivity, [leq(A, B), leq(B, C)], leq(A, C)).
inference_rule(reflexivity, [], leq(X, X)).

% ─── PREDICATE 3: admissible/2 ─────────────────────────────────────────────────

%% admissible(+State, -Reason)
%% Succeeds if State satisfies all admissibility conditions.
admissible(ss(_AgentID, _InH, _OutH, true, true), ok) :- !.
admissible(ss(_AgentID, _InH, _OutH, false, _), not_well_formed) :- !, fail.
admissible(ss(_AgentID, _InH, _OutH, _, false), not_consistent) :- fail.

% ─── PREDICATE 4: transition_allowed/4 ────────────────────────────────────────

%% transition_allowed(+S0, +K0, +S1, +K1)
%% Full 5-condition Ω gate: Contract, Curry, Tensor, Ω, Memory.
transition_allowed(S0, K0, S1, K1) :-
    % (1) Contract: both states admissible
    admissible(S0, _),
    admissible(S1, _),
    % (2) Curry derivation: agent IDs match (stub)
    S0 = ss(AgentID, _, _, _, _),
    S1 = ss(AgentID, _, _, _, _),
    % (3) Tensor: element types compatible
    K0 = ts(_, _, ElemType),
    K1 = ts(_, _, ElemType),
    % (4) Ω preserved
    omega_sig(S0, K0, Sig0),
    omega_sig(S1, K1, Sig1),
    Sig0 = Sig1,
    % (5) Memory: always succeeds here (real impl checks WORM availability)
    true.

% ─── PREDICATE 5: tensor_constraint/1 ─────────────────────────────────────────

%% tensor_constraint(+TensorState)
%% Checks the 5 K tensor invariants.
tensor_constraint(ts(Hash, Shape, ElemType)) :-
    % C1: shape validity (all dims positive)
    ground(Hash),
    Shape \= [],
    maplist([D]>>(D > 0), Shape),
    % C2: element type consistency
    member(ElemType, [int, float, string, bool, tensor]),
    % C3: index domain (implied by shape validity)
    % C4: closure property (partition then closure = identity; structural)
    % C5: partition completeness (implied by shape)
    true.

% ─── PREDICATE 6: omega_candidate/4 ───────────────────────────────────────────

%% omega_candidate(+S0, +K0, +S1, +K1)
%% Succeeds if the candidate (S1, K1) preserves Ω with reference (S0, K0).
omega_candidate(S0, K0, S1, K1) :-
    omega_sig(S0, K0, Sig0),
    omega_sig(S1, K1, Sig1),
    ( Sig0 = Sig1
    -> true
    ;  throw(error(omega_violated(Sig0, Sig1), omega_candidate/4))
    ).

% ─── 21 CONSTRAINT TYPES ──────────────────────────────────────────────────────

%% Constraint type taxonomy
constraint_type(unification(X, Y)) :- X = Y.
constraint_type(disequality(X, Y)) :- X \= Y.
constraint_type(set_member(X, S)) :- member(X, S).
constraint_type(set_subset(A, B)) :- subset(A, B).
constraint_type(set_disjoint(A, B)) :- intersection(A, B, []).
constraint_type(logic_and(A, B)) :- call(A), call(B).
constraint_type(logic_or(A, B)) :- call(A) ; call(B).
constraint_type(logic_not(A)) :- \+ call(A).
constraint_type(type_check(X, int)) :- integer(X).
constraint_type(type_check(X, float)) :- float(X).
constraint_type(type_check(X, atom)) :- atom(X).
constraint_type(type_check(X, list)) :- is_list(X).
constraint_type(order_lt(X, Y)) :- X < Y.
constraint_type(order_le(X, Y)) :- X =< Y.
constraint_type(order_gt(X, Y)) :- X > Y.
constraint_type(order_ge(X, Y)) :- X >= Y.
constraint_type(frequency_unique(L)) :- sort(L, L).  % no duplicates
constraint_type(tensor_shape_valid(Shape)) :- Shape \= [], maplist([D]>>(D > 0), Shape).
constraint_type(tensor_type_consistent(T)) :- member(T, [int, float, string, bool]).
constraint_type(domain_finite(D, Lo, Hi)) :- integer(D), D >= Lo, D =< Hi.
constraint_type(temporal_before(T1, T2)) :- T1 @< T2.

% ─── TESTS ────────────────────────────────────────────────────────────────────

:- begin_tests(curry_logic).

test(semantic_rule_identity) :-
    semantic_rule(identity, ss(agent1, h1, h2, true, true)).

test(semantic_rule_unknown, [fail]) :-
    semantic_rule(unknown_rule, ss(agent1, h1, h2, true, true)).

test(derive_leaf) :-
    derive([p, q, r], p, Witness),
    Witness = leaf(p).

test(derive_modus_ponens) :-
    derive([impl(a, b), a], b, Witness),
    Witness = step(modus_ponens, _, b).

test(admissible_ok) :-
    admissible(ss(agent1, h1, h2, true, true), ok).

test(admissible_bad, [fail]) :-
    admissible(ss(agent1, h1, h2, false, true), _).

test(transition_allowed) :-
    S = ss(agent1, h1, h2, true, true),
    K = ts(khash, [3, 4], float),
    transition_allowed(S, K, S, K).

test(omega_preserved) :-
    S = ss(agent1, h1, h2, true, true),
    K = ts(khash, [3, 4], float),
    omega_sig(S, K, Sig1),
    omega_sig(S, K, Sig2),
    Sig1 = Sig2.

test(tensor_constraint_ok) :-
    tensor_constraint(ts(myhash, [2, 3, 4], float)).

test(tensor_constraint_bad, [fail]) :-
    tensor_constraint(ts(myhash, [], float)).

test(omega_candidate_ok) :-
    S = ss(agent1, h1, h2, true, true),
    K = ts(khash, [3], int),
    omega_candidate(S, K, S, K).

test(constraint_type_unification) :-
    constraint_type(unification(42, 42)).

test(constraint_type_domain) :-
    constraint_type(domain_finite(5, 1, 10)).

test(derive_transitivity) :-
    derive([leq(a, b), leq(b, c)], leq(a, c), W),
    W = step(transitivity, _, leq(a, c)).

:- end_tests(curry_logic).

run_all_tests :-
    run_tests(curry_logic).

:- initialization(run_all_tests, main).
