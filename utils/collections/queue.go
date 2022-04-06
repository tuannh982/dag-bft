package collections

import (
	"fmt"
)

type Queue[V any] interface {
	Push(V)
	Pop() V
	Peek() V
	Size() int
}

type queue[V any] struct {
	entries []V
}

func NewQueue[V any]() Queue[V] {
	return &queue[V]{
		entries: make([]V, 0),
	}
}

func (s *queue[V]) Push(v V) {
	s.entries = append(s.entries, v)
}

func (s *queue[V]) Pop() (v V) {
	n := len(s.entries)
	if n == 0 {
		return v
	}
	ret := s.entries[0]
	s.entries = s.entries[1:]
	return ret
}

func (s *queue[V]) Peek() (v V) {
	n := len(s.entries)
	if n == 0 {
		return v
	}
	return s.entries[0]
}

func (s *queue[V]) Size() int {
	return len(s.entries)
}

func (s queue[V]) String() string {
	return fmt.Sprint(s.entries)
}
