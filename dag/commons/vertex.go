package commons

import (
	"fmt"
)

type Vertex struct {
	BaseVertex
	StrongEdges []BaseVertex
	WeakEdges   []BaseVertex
	Delivered   bool
}

type BaseVertex struct {
	Source Address
	Round  Round
	Block  Block
}

func Equals(a, b *Vertex) bool {
	if a == b {
		return true
	}
	return a.Source == b.Source && a.Round == b.Round && a.Block == b.Block
}

func (v *BaseVertex) Hash() string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%d-%s", v.Round, v.Source)
}

func VertexHash(v *Vertex) string {
	return v.Hash()
}

func (v *Vertex) StrongEdgesValues() []BaseVertex {
	return v.StrongEdges
}

func (v *Vertex) StrongEdgesContainsSource(a Address) bool {
	for _, u := range v.StrongEdges {
		if u.Source == a {
			return true
		}
	}
	return false
}

func (v *Vertex) WeakEdgesValues() []BaseVertex {
	return v.WeakEdges
}

func (v Vertex) String() string {
	strong := make([]string, 0, len(v.StrongEdges))
	for _, u := range v.StrongEdgesValues() {
		strong = append(strong, fmt.Sprintf("(s=%s,r=%s)", u.Source, u.Block))
	}
	weak := make([]string, 0, len(v.WeakEdges))
	for _, u := range v.WeakEdgesValues() {
		weak = append(weak, fmt.Sprintf("(s=%s,r=%d,b=%s)", u.Source, u.Round, u.Block))
	}
	return fmt.Sprintf("(s=%s,r=%d,b=%s,strong=%s,weak=%s)", v.Source, v.Round, v.Block, strong, weak)
}

func (v BaseVertex) String() string {
	return fmt.Sprintf("(s=%s,r=%d,b=%s)", v.Source, v.Round, v.Block)
}
