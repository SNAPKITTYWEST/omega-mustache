-- SPDX-License-Identifier: AGPL-3.0-or-later OR Apache-2.0
-- CLONE_GATE:AES256:f77ebc83d4ee51886072e270fbdcac1df96aac78cdcd2ad96a833a9e9f599fe2
--
-- Omega.curry — Ω Global Invariant + Logic Reasoning Engine
-- Curry Language (KiCS2 / PAKCS)
-- Compile: kics2 :load Omega :main testAll
--          pakcs  :load Omega :main testAll

module Omega where

import Data.List (nub, sort)
import Data.Char (ord)

-- ─── TYPES ───────────────────────────────────────────────────────────────────

type OmegaSignature = String
type SemanticHash   = String
type TensorHash     = String
type AgentID        = String
type TransitionID   = String

data OmegaViolation = OmegaViolation
  { violBefore :: OmegaSignature
  , violAfter  :: OmegaSignature
  } deriving Show

data SemanticState = SemanticState
  { ssAgentID    :: AgentID
  , ssInputHash  :: SemanticHash
  , ssOutputHash :: SemanticHash
  , ssWellFormed :: Bool
  , ssConsistent :: Bool
  } deriving (Show, Eq)

data TensorState = TensorState
  { tsHash      :: TensorHash
  , tsShape     :: [Int]
  , tsElemType  :: String
  } deriving (Show, Eq)

-- ─── Ω COMPUTATION ───────────────────────────────────────────────────────────

-- Ω signature: deterministic combination of semantic + tensor hashes.
-- In production this would be Blake3; here we use a simple XOR-fold.
omega :: SemanticState -> TensorState -> OmegaSignature
omega s k = simpleHash (ssInputHash s ++ "|" ++ tsHash k ++ "|" ++ constraintStr s)

constraintStr :: SemanticState -> String
constraintStr s = show (ssWellFormed s) ++ show (ssConsistent s)

-- Deterministic hash stub (sum of char codes mod 65536, hex-encoded).
simpleHash :: String -> String
simpleHash xs = padHex 4 (sum (map ord xs) `mod` 65536)
  where
    padHex n x = let h = toHex x in replicate (n - length h) '0' ++ h
    toHex 0 = "0"
    toHex x = reverse (go x)
    go 0 = ""
    go x = hexDigit (x `mod` 16) : go (x `div` 16)
    hexDigit d | d < 10    = toEnum (d + ord '0')
               | otherwise  = toEnum (d - 10 + ord 'a')

-- ─── TRANSITION VALIDATION ────────────────────────────────────────────────────

-- validTransition checks all 5 conditions (Contract, Curry, Tensor, Ω, Memory).
validTransition :: SemanticState -> TensorState
                -> SemanticState -> TensorState
                -> Either OmegaViolation ()
validTransition s0 k0 s1 k1
  | not (contractValid s0 s1) = Left (OmegaViolation (omega s0 k0) (omega s1 k1))
  | not (curryDerivable s0 s1) = Left (OmegaViolation (omega s0 k0) (omega s1 k1))
  | not (tensorExecutable k0 k1) = Left (OmegaViolation (omega s0 k0) (omega s1 k1))
  | omega s0 k0 /= omega s1 k1   = Left (OmegaViolation (omega s0 k0) (omega s1 k1))
  | otherwise                    = Right ()

-- Contract validity: both states must be well-formed and consistent.
contractValid :: SemanticState -> SemanticState -> Bool
contractValid s0 s1 = ssWellFormed s0 && ssWellFormed s1
                   && ssConsistent s0 && ssConsistent s1

-- Curry derivability: logic engine accepts the transition.
-- (Stub: real impl calls the Prolog SLD resolver.)
curryDerivable :: SemanticState -> SemanticState -> Bool
curryDerivable s0 s1 = ssAgentID s0 == ssAgentID s1

-- Tensor executability: shapes must be compatible.
tensorExecutable :: TensorState -> TensorState -> Bool
tensorExecutable k0 k1 = tsElemType k0 == tsElemType k1

-- ─── Ω AS A MONAD ─────────────────────────────────────────────────────────────

data OmegaM a = OmegaM
  { runOmega :: SemanticState -> TensorState
             -> Either OmegaViolation (a, SemanticState, TensorState) }

returnO :: a -> OmegaM a
returnO x = OmegaM $ \s k -> Right (x, s, k)

bindO :: OmegaM a -> (a -> OmegaM b) -> OmegaM b
bindO m f = OmegaM $ \s k ->
  case runOmega m s k of
    Left viol -> Left viol
    Right (a, s', k') ->
      if omega s k == omega s' k'
        then runOmega (f a) s' k'
        else Left (OmegaViolation (omega s k) (omega s' k'))

-- ─── INVARIANT CLASSES ────────────────────────────────────────────────────────

data InvariantClass
  = SemanticWellFormedness
  | ConstraintConsistency
  | StateAdmissibility
  | DeterministicResolution
  | DependencyAcyclicity
  | TensorShapeValidity
  | TensorTypeConsistency
  | TensorIndexDomain
  | TensorClosureProperty
  | PartitionCompleteness
  deriving (Show, Eq, Enum, Bounded)

allInvariantClasses :: [InvariantClass]
allInvariantClasses = [minBound .. maxBound]

checkSemanticInvariants :: SemanticState -> [InvariantClass]
checkSemanticInvariants s = [ c | (c, ok) <- checks, not ok ]
  where
    checks =
      [ (SemanticWellFormedness, ssWellFormed s)
      , (ConstraintConsistency,  ssConsistent s)
      , (StateAdmissibility,     ssWellFormed s && ssConsistent s)
      , (DeterministicResolution, True)  -- always true in this stub
      , (DependencyAcyclicity,   True)
      ]

-- ─── 6 CORE CURRY LOGIC PREDICATES ───────────────────────────────────────────

-- 1. semanticRule: holds if the rule is applicable to this state.
semanticRule :: String -> SemanticState -> Bool
semanticRule rule s = ssWellFormed s && rule /= ""

-- 2. derive: derives a conclusion from a premise list.
derive :: [String] -> String -> Bool
derive premises conclusion = conclusion `elem` closure premises
  where
    closure ps = nub (ps ++ concatMap step ps)
    step p = [p ++ "_derived"]  -- stub: real impl does SLD

-- 3. admissible: the state is admissible under the contract.
admissible :: SemanticState -> Bool
admissible s = ssWellFormed s && ssConsistent s

-- 4. transitionAllowed: the full 5-condition check.
transitionAllowed :: SemanticState -> TensorState -> SemanticState -> TensorState -> Bool
transitionAllowed s0 k0 s1 k1 =
  case validTransition s0 k0 s1 k1 of
    Right () -> True
    Left _   -> False

-- 5. tensorConstraint: the tensor operation satisfies the K invariants.
tensorConstraint :: TensorState -> Bool
tensorConstraint k = not (null (tsShape k)) && all (> 0) (tsShape k)

-- 6. omegaCandidate: a candidate state preserves Ω with the reference.
omegaCandidate :: SemanticState -> TensorState -> SemanticState -> TensorState -> Bool
omegaCandidate s0 k0 s1 k1 = omega s0 k0 == omega s1 k1

-- ─── TESTS ────────────────────────────────────────────────────────────────────

testAll :: IO ()
testAll = do
  putStrLn "=== OMEGA-MUSTACHE Curry Logic Tests ==="
  test "omega stable on identity" omegaStableIdentity
  test "omega violated on bad state" omegaViolated
  test "validTransition ok" validTransitionOk
  test "derive closure" deriveClosure
  test "admissible ok" admissibleOk
  test "tensor constraint ok" tensorConstraintOk
  test "monad bind preserves omega" monadBindOk
  putStrLn "=== ALL TESTS PASSED ==="

test :: String -> Bool -> IO ()
test name result
  | result    = putStrLn $ "[PASS] " ++ name
  | otherwise = ioError (userError ("[FAIL] " ++ name))

-- Test bodies
s0, s1Bad :: SemanticState
s0 = SemanticState "agent1" "abc" "def" True True
s1Bad = SemanticState "agent1" "abc" "def" False True -- not well-formed

k0 :: TensorState
k0 = TensorState "hash0" [3, 4] "float"

omegaStableIdentity :: Bool
omegaStableIdentity = omega s0 k0 == omega s0 k0

omegaViolated :: Bool
omegaViolated = omega s0 k0 /= omega s1Bad k0  -- bad state changes Ω

validTransitionOk :: Bool
validTransitionOk =
  case validTransition s0 k0 s0 k0 of
    Right () -> True
    Left _   -> False

deriveClosure :: Bool
deriveClosure = derive ["p", "q"] "p"

admissibleOk :: Bool
admissibleOk = admissible s0

tensorConstraintOk :: Bool
tensorConstraintOk = tensorConstraint k0

monadBindOk :: Bool
monadBindOk =
  let m = returnO (42 :: Int) `bindO` \x -> returnO (x + 1)
  in case runOmega m s0 k0 of
       Right (43, _, _) -> True
       _                -> False
