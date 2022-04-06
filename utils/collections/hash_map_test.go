package collections

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashMap(t *testing.T) {
	type Mock struct {
		A string
		B int
	}
	s := NewHashMap[string, *Mock]()
	_ = s.Put("aa", &Mock{
		A: "aa",
		B: 22,
	}, false)
	_ = s.Put("bb", &Mock{
		A: "bb",
		B: 55,
	}, false)
	require.Equal(t, 2, s.Size())
	require.Equal(t, true, s.Contains("aa"))
	require.Equal(t, true, s.Contains("bb"))
	require.Equal(t, false, s.Contains("cc"))
	require.Equal(t, 2, len(s.Keys()))
	require.Equal(t, 2, len(s.Values()))
	require.Nil(t, s.Delete("bb"))
	require.Equal(t, false, s.Contains("bb"))
	require.Equal(t, 1, s.Size())
}
