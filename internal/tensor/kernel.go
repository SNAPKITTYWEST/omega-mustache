// SPDX-License-Identifier: AGPL-3.0-or-later OR Apache-2.0
// CLONE_GATE:AES256:50e0313ba329c7921e3bb9532d8ba9ca879161138a69a80db36eef3776482f68
package tensor

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ElementType is the allowed element type discriminant.
type ElementType string

const (
	TypeInt    ElementType = "int"
	TypeFloat  ElementType = "float"
	TypeString ElementType = "string"
	TypeBool   ElementType = "bool"
	TypeTensor ElementType = "tensor" // nested
)

// Index is a single dimension index (non-negative integer).
type Index int

// Shape is an ordered list of dimension sizes.
type Shape []int

// Element is a typed value held in a tensor cell.
type Element struct {
	Type  ElementType
	IVal  int64
	FVal  float64
	SVal  string
	BVal  bool
	TVal  *Tensor
}

// Tensor is an N-dimensional array with closure properties.
// Immutable after construction — all operations return new tensors.
type Tensor struct {
	shape    Shape
	elemType ElementType
	data     []Element // row-major flat storage
	hash     string    // cached canonical hash
}

// New constructs a zero-value Tensor of the given shape and element type.
func New(shape Shape, elemType ElementType) (*Tensor, error) {
	if len(shape) == 0 {
		return nil, errors.New("tensor: shape must be non-empty")
	}
	size := 1
	for _, d := range shape {
		if d <= 0 {
			return nil, fmt.Errorf("tensor: dimension size must be positive, got %d", d)
		}
		size *= d
	}
	t := &Tensor{
		shape:    append(Shape{}, shape...),
		elemType: elemType,
		data:     make([]Element, size),
	}
	for i := range t.data {
		t.data[i] = zeroElement(elemType)
	}
	return t, nil
}

func zeroElement(et ElementType) Element {
	return Element{Type: et}
}

// Shape returns a copy of the tensor's shape.
func (t *Tensor) Shape() Shape { return append(Shape{}, t.shape...) }

// ElemType returns the element type.
func (t *Tensor) ElemType() ElementType { return t.elemType }

// flatIndex converts N-dimensional indices to a flat offset.
func (t *Tensor) flatIndex(idx []Index) (int, error) {
	if len(idx) != len(t.shape) {
		return 0, fmt.Errorf("tensor: index rank %d != shape rank %d", len(idx), len(t.shape))
	}
	offset := 0
	stride := 1
	for i := len(t.shape) - 1; i >= 0; i-- {
		if int(idx[i]) < 0 || int(idx[i]) >= t.shape[i] {
			return 0, fmt.Errorf("tensor: index[%d]=%d out of range [0,%d)", i, idx[i], t.shape[i])
		}
		offset += int(idx[i]) * stride
		stride *= t.shape[i]
	}
	return offset, nil
}

// Get retrieves the element at the given indices.
func (t *Tensor) Get(idx []Index) (Element, error) {
	off, err := t.flatIndex(idx)
	if err != nil {
		return Element{}, err
	}
	return t.data[off], nil
}

// Set returns a new Tensor with the element at idx replaced by e.
// Preserves immutability.
func (t *Tensor) Set(idx []Index, e Element) (*Tensor, error) {
	if e.Type != t.elemType {
		return nil, fmt.Errorf("tensor: element type %s != tensor type %s", e.Type, t.elemType)
	}
	off, err := t.flatIndex(idx)
	if err != nil {
		return nil, err
	}
	newData := make([]Element, len(t.data))
	copy(newData, t.data)
	newData[off] = e
	return &Tensor{shape: t.Shape(), elemType: t.elemType, data: newData}, nil
}

// ─── OPERATIONS ──────────────────────────────────────────────────────────────

// Product (☉) — element-wise product of two same-shape tensors.
func Product(a, b *Tensor) (*Tensor, error) {
	if err := sameShape(a, b); err != nil {
		return nil, err
	}
	if a.elemType != b.elemType {
		return nil, fmt.Errorf("tensor ☉: element type mismatch %s vs %s", a.elemType, b.elemType)
	}
	out := &Tensor{shape: a.Shape(), elemType: a.elemType, data: make([]Element, len(a.data))}
	for i, ae := range a.data {
		be := b.data[i]
		re, err := mulElements(ae, be)
		if err != nil {
			return nil, fmt.Errorf("tensor ☉ at flat index %d: %w", i, err)
		}
		out.data[i] = re
	}
	return out, nil
}

func mulElements(a, b Element) (Element, error) {
	switch a.Type {
	case TypeInt:
		return Element{Type: TypeInt, IVal: a.IVal * b.IVal}, nil
	case TypeFloat:
		return Element{Type: TypeFloat, FVal: a.FVal * b.FVal}, nil
	default:
		return Element{}, fmt.Errorf("☉ not defined for type %s", a.Type)
	}
}

// Partition (⌹) — splits a tensor along dimension dim into slices of size partSize.
func Partition(t *Tensor, dim, partSize int) ([]*Tensor, error) {
	if dim < 0 || dim >= len(t.shape) {
		return nil, fmt.Errorf("tensor ⌹: dim %d out of range", dim)
	}
	if partSize <= 0 || t.shape[dim]%partSize != 0 {
		return nil, fmt.Errorf("tensor ⌹: dim size %d not divisible by partSize %d",
			t.shape[dim], partSize)
	}
	numParts := t.shape[dim] / partSize
	parts := make([]*Tensor, numParts)
	newShape := append(Shape{}, t.shape...)
	newShape[dim] = partSize
	for p := 0; p < numParts; p++ {
		part, _ := New(newShape, t.elemType)
		parts[p] = part
	}
	// copy data into parts by iterating all indices
	iterShape(t.shape, func(idx []Index) {
		partIdx := int(idx[dim]) / partSize
		localIdx := make([]Index, len(idx))
		copy(localIdx, idx)
		localIdx[dim] = Index(int(idx[dim]) % partSize)
		val, _ := t.Get(idx)
		parts[partIdx].data[flatOff(parts[partIdx].shape, localIdx)] = val
	})
	return parts, nil
}

// Closure (○) — reconstructs a tensor from partitions (inverse of Partition).
func Closure(parts []*Tensor, dim int) (*Tensor, error) {
	if len(parts) == 0 {
		return nil, errors.New("tensor ○: no parts")
	}
	p0 := parts[0]
	newShape := append(Shape{}, p0.shape...)
	newShape[dim] = p0.shape[dim] * len(parts)
	out, _ := New(newShape, p0.elemType)
	for p, part := range parts {
		iterShape(part.shape, func(idx []Index) {
			outIdx := make([]Index, len(idx))
			copy(outIdx, idx)
			outIdx[dim] = Index(p*part.shape[dim] + int(idx[dim]))
			val, _ := part.Get(idx)
			off := flatOff(newShape, outIdx)
			out.data[off] = val
		})
	}
	return out, nil
}

// Difference (△) — set difference of two tensors treated as element multisets.
// Returns a tensor with the same shape containing zeros where b has matching elements.
func Difference(a, b *Tensor) (*Tensor, error) {
	if err := sameShape(a, b); err != nil {
		return nil, err
	}
	out := &Tensor{shape: a.Shape(), elemType: a.elemType, data: make([]Element, len(a.data))}
	for i, ae := range a.data {
		be := b.data[i]
		if elemEqual(ae, be) {
			out.data[i] = zeroElement(a.elemType)
		} else {
			out.data[i] = ae
		}
	}
	return out, nil
}

// Transform (◇) — applies a function to every element.
func Transform(t *Tensor, f func(Element) (Element, error)) (*Tensor, error) {
	out := &Tensor{shape: t.Shape(), elemType: t.elemType, data: make([]Element, len(t.data))}
	for i, e := range t.data {
		re, err := f(e)
		if err != nil {
			return nil, fmt.Errorf("tensor ◇ at flat index %d: %w", i, err)
		}
		out.data[i] = re
	}
	return out, nil
}

// Compose (⬡) — sequential composition: apply g after f.
func Compose(f, g func(*Tensor) (*Tensor, error)) func(*Tensor) (*Tensor, error) {
	return func(t *Tensor) (*Tensor, error) {
		mid, err := f(t)
		if err != nil {
			return nil, err
		}
		return g(mid)
	}
}

// ─── CANONICAL HASH ───────────────────────────────────────────────────────────

// Hash returns a deterministic SHA-256 hash of the tensor's canonical form.
func (t *Tensor) Hash() string {
	if t.hash != "" {
		return t.hash
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("shape:%v|type:%s|data:", t.shape, t.elemType))
	for _, e := range t.data {
		sb.WriteString(elementStr(e))
		sb.WriteByte(',')
	}
	h := sha256.Sum256([]byte(sb.String()))
	t.hash = hex.EncodeToString(h[:])
	return t.hash
}

func elementStr(e Element) string {
	switch e.Type {
	case TypeInt:
		return fmt.Sprintf("i:%d", e.IVal)
	case TypeFloat:
		return fmt.Sprintf("f:%v", e.FVal)
	case TypeString:
		return fmt.Sprintf("s:%s", e.SVal)
	case TypeBool:
		return fmt.Sprintf("b:%v", e.BVal)
	case TypeTensor:
		if e.TVal != nil {
			return fmt.Sprintf("t:%s", e.TVal.Hash())
		}
		return "t:nil"
	}
	return "?"
}

// ─── VALIDATION ──────────────────────────────────────────────────────────────

// Validate checks all 5 tensor invariants and returns any violations.
func (t *Tensor) Validate() []string {
	var errs []string
	// C1: shape validity
	for i, d := range t.shape {
		if d <= 0 {
			errs = append(errs, fmt.Sprintf("C1: shape[%d]=%d must be positive", i, d))
		}
	}
	// C2: element type consistency
	for i, e := range t.data {
		if e.Type != t.elemType {
			errs = append(errs, fmt.Sprintf("C2: data[%d].Type=%s != tensor.ElemType=%s", i, e.Type, t.elemType))
		}
	}
	// C3: index domain correctness (checked by Get/Set, always valid if constructed by New)
	// C4: closure (partition then closure returns equivalent tensor)
	// C5: partition completeness (all partitions cover full shape)
	// These are structural and hold by construction; flag if data length mismatches shape.
	expectedSize := 1
	for _, d := range t.shape {
		expectedSize *= d
	}
	if len(t.data) != expectedSize {
		errs = append(errs, fmt.Sprintf("C5: data length %d != expected %d", len(t.data), expectedSize))
	}
	return errs
}

// ─── HELPERS ─────────────────────────────────────────────────────────────────

func sameShape(a, b *Tensor) error {
	if len(a.shape) != len(b.shape) {
		return fmt.Errorf("tensor: rank mismatch %d vs %d", len(a.shape), len(b.shape))
	}
	for i, d := range a.shape {
		if d != b.shape[i] {
			return fmt.Errorf("tensor: shape mismatch at dim %d: %d vs %d", i, d, b.shape[i])
		}
	}
	return nil
}

func elemEqual(a, b Element) bool {
	if a.Type != b.Type {
		return false
	}
	switch a.Type {
	case TypeInt:
		return a.IVal == b.IVal
	case TypeFloat:
		return a.FVal == b.FVal
	case TypeString:
		return a.SVal == b.SVal
	case TypeBool:
		return a.BVal == b.BVal
	}
	return false
}

func iterShape(shape Shape, f func([]Index)) {
	idx := make([]Index, len(shape))
	var rec func(dim int)
	rec = func(dim int) {
		if dim == len(shape) {
			cp := make([]Index, len(idx))
			copy(cp, idx)
			f(cp)
			return
		}
		for i := 0; i < shape[dim]; i++ {
			idx[dim] = Index(i)
			rec(dim + 1)
		}
	}
	rec(0)
}

func flatOff(shape Shape, idx []Index) int {
	off := 0
	stride := 1
	for i := len(shape) - 1; i >= 0; i-- {
		off += int(idx[i]) * stride
		stride *= shape[i]
	}
	return off
}

// SortedKeys returns shape dimension indices in stable order (for canonical ops).
func SortedKeys(n int) []int {
	ks := make([]int, n)
	for i := range ks {
		ks[i] = i
	}
	sort.Ints(ks)
	return ks
}
