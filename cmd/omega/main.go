// SPDX-License-Identifier: AGPL-3.0-or-later OR Apache-2.0
// CLONE_GATE:AES256:f77ebc83d4ee51886072e270fbdcac1df96aac78cdcd2ad96a833a9e9f599fe2
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/SNAPKITTYWEST/omega-mustache/internal/contract"
	"github.com/SNAPKITTYWEST/omega-mustache/internal/handler"
	"github.com/SNAPKITTYWEST/omega-mustache/internal/worm"
)

func main() {
	addr := os.Getenv("OMEGA_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	// Build component graph
	chain := worm.New()
	registry := contract.NewRegistry()

	// Register built-in contracts
	if err := registerBuiltinContracts(registry); err != nil {
		log.Fatalf("failed to register contracts: %v", err)
	}

	h := handler.New(registry, chain)

	mux := http.NewServeMux()
	mux.Handle("/transition", h)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		n := chain.Len()
		fmt.Fprintf(w, `{"status":"ok","worm_len":%d}`, n)
	})
	mux.HandleFunc("/chain", func(w http.ResponseWriter, r *http.Request) {
		records := chain.Records()
		w.Header().Set("Content-Type", "application/json")
		for _, rec := range records {
			fmt.Fprintf(w, "%+v\n", rec)
		}
	})

	log.Printf("OMEGA-MUSTACHE listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func registerBuiltinContracts(r *contract.Registry) error {
	// Default identity contract: state is passed through unchanged.
	identity := &contract.Contract{
		ID:      "identity",
		Version: "1.0.0",
		Preconditions: []contract.Condition{
			{
				Name: "agent_id_present",
				Fn:   func(b, a contract.State) bool { return b.AgentID != "" },
			},
			{
				Name: "operation_present",
				Fn:   func(b, a contract.State) bool { return b.OperationName != "" },
			},
		},
		Postconditions: []contract.Condition{
			{
				Name: "output_hash_set",
				Fn:   func(b, a contract.State) bool { return a.OutputHash != "" },
			},
		},
		Invariants: []contract.Condition{
			{
				Name: "well_formed",
				Fn:   func(b, a contract.State) bool { return b.Flags.WellFormed },
			},
		},
		AdmissibilityFn: func(b, a contract.State) bool {
			return b.Flags.WellFormed && b.Flags.Consistent
		},
	}

	// Transform contract: permits deterministic state changes.
	transform := &contract.Contract{
		ID:      "transform",
		Version: "1.0.0",
		Preconditions: []contract.Condition{
			{
				Name: "input_hash_present",
				Fn:   func(b, a contract.State) bool { return b.InputHash != "" },
			},
			{
				Name: "deterministic",
				Fn:   func(b, a contract.State) bool { return b.Flags.Deterministic },
			},
		},
		Postconditions: []contract.Condition{
			{
				Name: "output_differs_or_noop",
				Fn:   func(b, a contract.State) bool { return true },
			},
		},
		Invariants: []contract.Condition{
			{
				Name: "acyclic_dependencies",
				Fn:   func(b, a contract.State) bool { return b.Flags.Acyclic },
			},
		},
		AdmissibilityFn: func(b, a contract.State) bool {
			return b.Flags.Deterministic && b.Flags.Consistent
		},
	}

	for _, c := range []*contract.Contract{identity, transform} {
		if err := r.Register(c); err != nil {
			return err
		}
	}
	return nil
}
