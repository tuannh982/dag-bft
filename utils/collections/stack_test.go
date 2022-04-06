package collections

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStack(t *testing.T) {
	type Mock struct {
		A string
		B int
	}
	Equals := func(a, b *Mock) bool {
		if a == b {
			return true
		} else {
			if a.A == b.A && a.B == b.B {
				return true
			} else {
				return false
			}
		}
	}
	s := NewStack[*Mock]()
	s.Push(&Mock{
		A: "aa",
		B: 22,
	})
	s.Push(&Mock{
		A: "bb",
		B: 55,
	})
	require.Equal(t, 2, s.Size())
	top := s.Pop()
	require.Equal(t, true, Equals(top, &Mock{
		A: "bb",
		B: 55,
	}))
	require.Equal(t, 1, s.Size())
	peek := s.Peek()
	require.Equal(t, true, Equals(peek, &Mock{
		A: "aa",
		B: 22,
	}))
	top = s.Pop()
	require.Equal(t, true, Equals(top, &Mock{
		A: "aa",
		B: 22,
	}))
	require.Equal(t, 0, s.Size())
}
