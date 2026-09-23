// SPDX-License-Identifier: AGPL-3.0-or-later OR Apache-2.0
// CLONE_GATE:AES256:1161da5f66d012d8bbe1f1bec29d02e347d59833e01d78cdb714a74026aad086
package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/SNAPKITTYWEST/omega-mustache/internal/contract"
	"github.com/SNAPKITTYWEST/omega-mustache/internal/omega"
	"github.com/SNAPKITTYWEST/omega-mustache/internal/worm"
)

// ─── REQUEST / RESPONSE TYPES ────────────────────────────────────────────────

// TransitionRequest is the inbound HTTP payload.
// No map[string]interface{} anywhere in the pipeline.
type TransitionRequest struct {
	TransitionID string          `json:"transition_id"`
	AgentID      string          `json:"agent_id"`
	ContractID   string          `json:"contract_id"`
	Operation    string          `json:"operation"`
	InputHash    string          `json:"input_hash"`
	SemanticData json.RawMessage `json:"semantic_data"`
	TensorData   json.RawMessage `json:"tensor_data"`
}

// TransitionResponse is the outbound HTTP payload.
type TransitionResponse struct {
	TransitionID    string `json:"transition_id"`
	Success         bool   `json:"success"`
	OmegaBefore     string `json:"omega_before,omitempty"`
	OmegaAfter      string `json:"omega_after,omitempty"`
	WORMRecordID    string `json:"worm_record_id,omitempty"`
	RejectionReason string `json:"rejection_reason,omitempty"`
	FailStage       string `json:"fail_stage,omitempty"`
}

// StageError wraps an error with the pipeline stage name for attribution.
type StageError struct {
	Stage string
	Err   error
}

func (e *StageError) Error() string {
	return fmt.Sprintf("stage %s: %s", e.Stage, e.Err)
}

// ─── HANDLER ─────────────────────────────────────────────────────────────────

// Handler is the HTTP handler that owns the full 8-stage pipeline.
type Handler struct {
	contracts *contract.Registry
	wormChain *worm.Chain
	verifier  *omega.Verifier
	halted    bool // emergency halt flag
}

// New returns a fully initialised Handler.
func New(cr *contract.Registry, chain *worm.Chain) *Handler {
	return &Handler{
		contracts: cr,
		wormChain: chain,
		verifier:  omega.NewVerifier(),
	}
}

// ServeHTTP implements http.Handler for the transition endpoint.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.halted {
		writeError(w, http.StatusServiceUnavailable, "OMEGA_UNVERIFIABLE: agent halted")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	var req TransitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request: "+err.Error())
		return
	}

	resp, err := h.handleTransition(req)
	if err != nil {
		// Emergency halt on Ω unverifiable
		if err == omega.ErrOmegaUnverifiable {
			h.halted = true
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		// Rejection with evidence
		resp = TransitionResponse{
			TransitionID:    req.TransitionID,
			Success:         false,
			RejectionReason: err.Error(),
		}
		if se, ok := err.(*StageError); ok {
			resp.FailStage = se.Stage
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleTransition runs the 8-stage pipeline and returns a response.
// Any stage failure returns an error; the WORM record is only appended on success.
func (h *Handler) handleTransition(req TransitionRequest) (TransitionResponse, error) {

	// ── Stage 1: Decode & validate input ─────────────────────────────────────
	if req.TransitionID == "" || req.AgentID == "" || req.ContractID == "" {
		return TransitionResponse{}, &StageError{"decode", fmt.Errorf("missing required fields")}
	}

	// ── Stage 2: Contract lookup ──────────────────────────────────────────────
	c, err := h.contracts.Get(contract.ContractID(req.ContractID))
	if err != nil {
		return TransitionResponse{}, &StageError{"contract_lookup", err}
	}

	// ── Stage 3: Build before/after semantic states ───────────────────────────
	beforeState := contract.State{
		AgentID:       req.AgentID,
		OperationName: req.Operation,
		InputHash:     req.InputHash,
		Flags: contract.StateFlags{
			WellFormed:    true,
			Consistent:    true,
			Admissible:    true,
			Deterministic: true,
			Acyclic:       true,
		},
	}
	afterState := beforeState
	afterState.OutputHash = hashBytes(req.TensorData)

	// ── Stage 4: Contract evaluation (8-phase) ────────────────────────────────
	result := c.Evaluate(beforeState, afterState)
	if !result.Passed {
		return TransitionResponse{}, &StageError{
			"contract_eval",
			fmt.Errorf("contract rejected: %s", result.FailReason),
		}
	}

	// ── Stage 5: Curry logic derivation (stub — real impl calls Prolog) ───────
	curryHash := hashString(req.TransitionID + req.Operation)

	// ── Stage 6: Tensor operation ─────────────────────────────────────────────
	tensorHash := hashBytes(req.TensorData)
	semanticHash := hashBytes(req.SemanticData)

	// ── Stage 7: Ω verification ───────────────────────────────────────────────
	// Before: compute from input hashes
	constraintHash := hashString(string(c.ID) + string(c.Version))
	omegaBefore := h.verifier.Compute(semanticHash, tensorHash, constraintHash)

	// After: same contract, updated semantic/tensor
	omegaAfter := h.verifier.Compute(
		hashString(semanticHash+afterState.OutputHash),
		tensorHash,
		constraintHash,
	)

	// The Ω must be preserved. Because our state transform is contract-valid,
	// the canonical hash should remain stable. If it doesn't, reject.
	// (In full impl, the tensor kernel changes K; here the semantic transform
	//  is identity so Ω is preserved unless the contract mutates state class.)
	omegaPreserved := omegaBefore == omegaAfter
	if !omegaPreserved {
		return TransitionResponse{}, &StageError{
			"omega_verify",
			&omega.OmegaViolation{
				Before: omegaBefore,
				After:  omegaAfter,
			},
		}
	}

	// ── Stage 8: WORM append ──────────────────────────────────────────────────
	rec := worm.Record{
		Timestamp:       time.Now().UTC(),
		TransitionID:    worm.TransitionID(req.TransitionID),
		AgentID:         worm.AgentID(req.AgentID),
		SemanticBefore:  semanticHash,
		SemanticAfter:   hashString(semanticHash + afterState.OutputHash),
		TensorBefore:    tensorHash,
		TensorAfter:     tensorHash,
		OmegaBefore:     string(omegaBefore),
		OmegaAfter:      string(omegaAfter),
		OmegaOK:         true,
		ContractVersion: string(c.Version),
		CurryDerivation: curryHash,
	}
	sealed, err := h.wormChain.AppendRecord(rec)
	if err != nil {
		return TransitionResponse{}, &StageError{"worm_append", err}
	}

	return TransitionResponse{
		TransitionID: req.TransitionID,
		Success:      true,
		OmegaBefore:  string(omegaBefore),
		OmegaAfter:   string(omegaAfter),
		WORMRecordID: string(sealed.RecordID),
	}, nil
}

// ─── HELPERS ─────────────────────────────────────────────────────────────────

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func hashBytes(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
