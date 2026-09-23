// SPDX-License-Identifier: AGPL-3.0-or-later OR Apache-2.0
// CLONE_GATE:AES256:8735aeae1b3fa5e9046dee7b0130bc7b181a28699211a50cd4bb089e2c786742
package render

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"time"
)

// ─── RENDER MODEL TYPES ───────────────────────────────────────────────────────
// All typed — no map[string]interface{}.

// TransitionModel is the top-level render model for a transition view.
type TransitionModel struct {
	Title        string
	TransitionID string
	AgentID      string
	Timestamp    time.Time
	Success      bool
	OmegaBefore  string
	OmegaAfter   string
	WORMRecord   WORMRecordModel
	Stages       []StageModel
	Error        ErrorModel
}

// WORMRecordModel is the render model for a WORM chain record.
type WORMRecordModel struct {
	RecordID     string
	PreviousHash string
	Hash         string
	OmegaOK      bool
	Timestamp    time.Time
}

// StageModel is the render model for a single pipeline stage.
type StageModel struct {
	Name    string
	Passed  bool
	Message string
}

// ErrorModel is the render model for rejection evidence.
type ErrorModel struct {
	Present  bool
	Stage    string
	Reason   string
	Evidence []StageModel
}

// OmegaStatusModel is the render model for the Ω status panel.
type OmegaStatusModel struct {
	Preserved bool
	Before    string
	After     string
	Classes   []InvariantClassModel
}

// InvariantClassModel is the render model for one invariant class check.
type InvariantClassModel struct {
	Name   string
	Passed bool
}

// ChainModel is the render model for the full WORM chain view.
type ChainModel struct {
	Title   string
	Length  int
	Records []WORMRecordModel
}

// TensorModel is the render model for a tensor state.
type TensorModel struct {
	Shape    []int
	ElemType string
	Hash     string
	// Data is a flat list of element strings for display
	Data []string
}

// ─── RENDERER ─────────────────────────────────────────────────────────────────

// Renderer manages Go html/template rendering for OMEGA-MUSTACHE views.
type Renderer struct {
	templates map[string]*template.Template
}

// NewRenderer parses all templates from the given directory glob.
func NewRenderer(templateDir string) (*Renderer, error) {
	r := &Renderer{templates: make(map[string]*template.Template)}

	patterns := []string{
		"transition.html",
		"omega_status.html",
		"chain.html",
		"tensor.html",
		"error.html",
		"index.html",
	}

	for _, pat := range patterns {
		path := filepath.Join(templateDir, pat)
		tmpl, err := template.ParseFiles(path)
		if err != nil {
			// Non-fatal at startup: template file may not exist yet
			continue
		}
		r.templates[pat] = tmpl
	}
	return r, nil
}

// RenderTransition renders the transition view.
func (r *Renderer) RenderTransition(w io.Writer, m TransitionModel) error {
	return r.render(w, "transition.html", m)
}

// RenderOmegaStatus renders the Ω status panel.
func (r *Renderer) RenderOmegaStatus(w io.Writer, m OmegaStatusModel) error {
	return r.render(w, "omega_status.html", m)
}

// RenderChain renders the WORM chain view.
func (r *Renderer) RenderChain(w io.Writer, m ChainModel) error {
	return r.render(w, "chain.html", m)
}

// RenderTensor renders the tensor display.
func (r *Renderer) RenderTensor(w io.Writer, m TensorModel) error {
	return r.render(w, "tensor.html", m)
}

func (r *Renderer) render(w io.Writer, name string, data any) error {
	tmpl, ok := r.templates[name]
	if !ok {
		return fmt.Errorf("render: template %q not loaded", name)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("render: execute %q failed: %w", name, err)
	}
	_, err := w.Write(buf.Bytes())
	return err
}
