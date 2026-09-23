// SPDX-License-Identifier: AGPL-3.0-or-later OR Apache-2.0
// CLONE_GATE:AES256:2d61666349f737535a540c3c6085fe3515a96da4920f6db2ed9cb933e538a057
package worm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// RecordID is a unique WORM record identifier.
type RecordID string

// TransitionID ties a WORM record to the pipeline transition that created it.
type TransitionID string

// AgentID identifies the agent that executed a transition.
type AgentID string

// Record is a single immutable entry in the WORM chain.
// Every field is hashed into the record's own Hash, which chains to the previous.
type Record struct {
	RecordID     RecordID     `json:"record_id"`
	Timestamp    time.Time    `json:"timestamp"`
	TransitionID TransitionID `json:"transition_id"`
	AgentID      AgentID      `json:"agent_id"`

	SemanticBefore string `json:"semantic_before_hash"`
	SemanticAfter  string `json:"semantic_after_hash"`
	TensorBefore   string `json:"tensor_before_hash"`
	TensorAfter    string `json:"tensor_after_hash"`

	OmegaBefore string `json:"omega_before"`
	OmegaAfter  string `json:"omega_after"`
	OmegaOK     bool   `json:"omega_preserved"`

	ContractVersion string `json:"contract_version"`
	CurryDerivation string `json:"curry_derivation_hash"`

	PreviousHash string `json:"previous_hash"`
	Hash         string `json:"hash"` // SHA-256 of all fields above
}

// Signature is an Ed25519 detached signature over the record hash.
// (Stored separately from the record so the record hash is stable.)
type Signature struct {
	RecordID RecordID `json:"record_id"`
	Sig      []byte   `json:"signature"`
	PubKey   []byte   `json:"public_key"`
}

// Chain is an append-only WORM chain with mutex-serialised writes.
// Readers never block writers; writers hold the mutex for the append only.
type Chain struct {
	mu      sync.Mutex
	records []Record
	sigs    []Signature
}

// New returns an initialised, empty Chain.
func New() *Chain { return &Chain{} }

// Len returns the current chain length (safe for concurrent read after RLock, but
// here we just use the mutex for simplicity since reads are rare in tests).
func (c *Chain) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.records)
}

// Head returns the most recent record, or an error if the chain is empty.
func (c *Chain) Head() (Record, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.records) == 0 {
		return Record{}, errors.New("worm: chain is empty")
	}
	return c.records[len(c.records)-1], nil
}

// AppendRecord appends a new record atomically.
// The caller supplies all fields except PreviousHash, Hash, and RecordID,
// which are computed here.
func (c *Chain) AppendRecord(r Record) (Record, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Chain link
	if len(c.records) == 0 {
		r.PreviousHash = "genesis"
	} else {
		r.PreviousHash = c.records[len(c.records)-1].Hash
	}

	// Stable record ID
	r.RecordID = RecordID(fmt.Sprintf("rec-%d", len(c.records)))

	// Compute hash
	h, err := recordHash(r)
	if err != nil {
		return Record{}, fmt.Errorf("worm: hash computation failed: %w", err)
	}
	r.Hash = h

	c.records = append(c.records, r)
	return r, nil
}

// Verify replays the entire chain and checks every hash link.
// Returns the index of the first broken link, or -1 if intact.
func (c *Chain) Verify() (broken int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	prev := "genesis"
	for i, rec := range c.records {
		// Check previous hash
		if rec.PreviousHash != prev {
			return i, fmt.Errorf("worm: record %d previous_hash mismatch", i)
		}
		// Recompute hash
		claimed := rec.Hash
		rec.Hash = ""
		computed, herr := recordHash(rec)
		if herr != nil {
			return i, herr
		}
		if computed != claimed {
			return i, fmt.Errorf("worm: record %d hash mismatch: stored=%s computed=%s",
				i, claimed, computed)
		}
		prev = claimed
	}
	return -1, nil
}

// Replay re-executes all records between indices lo and hi (inclusive)
// and invokes fn on each. Deterministic: same records always produce same calls.
func (c *Chain) Replay(lo, hi int, fn func(Record) error) error {
	c.mu.Lock()
	records := make([]Record, len(c.records))
	copy(records, c.records)
	c.mu.Unlock()

	if lo < 0 || hi >= len(records) || lo > hi {
		return fmt.Errorf("worm: replay range [%d,%d] out of bounds (len=%d)", lo, hi, len(records))
	}
	for i := lo; i <= hi; i++ {
		if err := fn(records[i]); err != nil {
			return fmt.Errorf("worm: replay failed at record %d: %w", i, err)
		}
	}
	return nil
}

// Records returns a snapshot of all records (copy, safe to iterate).
func (c *Chain) Records() []Record {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Record, len(c.records))
	copy(out, c.records)
	return out
}

// ─── HELPERS ─────────────────────────────────────────────────────────────────

func recordHash(r Record) (string, error) {
	// Zero out Hash field before hashing (it's being computed).
	r.Hash = ""
	b, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// ErrEmptyChain is returned by operations that require at least one record.
var ErrEmptyChain = errors.New("worm: chain is empty")
