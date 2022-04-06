package collections

type hashSet[R comparable, V any] struct {
	entries  map[R]V
	hashFunc HashSetHashFunc[R, V]
}

type HashSetHashFunc[R comparable, V any] func(V) R

func NewHashSet[R comparable, V any](f HashSetHashFunc[R, V]) Set[V] {
	return &hashSet[R, V]{
		entries:  make(map[R]V),
		hashFunc: f,
	}
}

func (s *hashSet[R, V]) Contains(v V) bool {
	hash := s.hashFunc(v)
	if _, ok := s.entries[hash]; ok {
		return true
	}
	return false
}

func (s *hashSet[R, V]) Add(v V) error {
	if s.Contains(v) {
		return ErrValueExisted
	}
	s.entries[s.hashFunc(v)] = v
	return nil
}

func (s *hashSet[R, V]) Remove(v V) error {
	if !s.Contains(v) {
		return ErrValueNotExisted
	}
	delete(s.entries, s.hashFunc(v))
	return nil
}

func (s *hashSet[R, V]) Size() int {
	return len(s.entries)
}

func (s *hashSet[R, V]) Entries() []V {
	arr := make([]V, 0, s.Size())
	for _, v := range s.entries {
		arr = append(arr, v)
	}
	return arr
}
