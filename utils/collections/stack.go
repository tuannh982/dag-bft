package collections

import "fmt"

type Stack[V any] interface {
	Push(V)
	Pop() V
	Peek() V
	Size() int
}

type stack[V any] struct {
	entries []V
}

func NewStack[V any]() Stack[V] {
	return &stack[V]{
		entries: make([]V, 0),
	}
}

func (s *stack[V]) Push(v V) {
	s.entries = append(s.entries, v)
}

func (s *stack[V]) Pop() (v V) {
	n := len(s.entries)
	if n == 0 {
		return v
	}
	ret := s.entries[n-1]
	s.entries = s.entries[:n-1]
	return ret
}

func (s *stack[V]) Peek() (v V) {
	n := len(s.entries)
	if n == 0 {
		return v
	}
	return s.entries[n-1]
}

func (s *stack[V]) Size() int {
	return len(s.entries)
}

func (s stack[V]) String() string {
	return fmt.Sprint(s.entries)
}
