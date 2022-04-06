package collections

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashSet(t *testing.T) {
	type Mock struct {
		A string
		B int
	}
	s := NewHashSet(func(v *Mock) string {
		return v.A
	})
	require.Nil(t, s.Add(&Mock{
		A: "aa",
		B: 22,
	}))
	require.NotNil(t, s.Add(&Mock{
		A: "aa",
		B: 22,
	}))
	require.Nil(t, s.Add(&Mock{
		A: "bb",
		B: 55,
	}))
	require.Equal(t, 2, s.Size())
	require.Equal(t, true, s.Contains(&Mock{
		A: "aa",
	}))
	require.Equal(t, true, s.Contains(&Mock{
		A: "bb",
	}))
	require.Equal(t, false, s.Contains(&Mock{
		A: "cc",
	}))
	require.Equal(t, 2, len(s.Entries()))
	require.Nil(t, s.Remove(&Mock{
		A: "bb",
	}))
	require.Equal(t, false, s.Contains(&Mock{
		A: "bb",
	}))
	require.Equal(t, 1, s.Size())
}
