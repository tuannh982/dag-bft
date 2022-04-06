package collections

type hashMap[K comparable, V any] struct {
	entries map[K]V
}

func NewHashMap[K comparable, V any]() Map[K, V] {
	return &hashMap[K, V]{
		entries: make(map[K]V),
	}
}

func (m *hashMap[K, V]) Contains(k K) bool {
	if _, ok := m.entries[k]; ok {
		return true
	}
	return false
}

func (m *hashMap[K, V]) Put(k K, v V, forced bool) error {
	if forced {
		m.entries[k] = v
		return nil
	}
	if m.Contains(k) {
		return ErrValueExisted
	}
	m.entries[k] = v
	return nil
}

func (m *hashMap[K, V]) Get(k K) (v V, err error) {
	if !m.Contains(k) {
		return v, ErrValueNotExisted
	}
	return m.entries[k], nil
}

func (m *hashMap[K, V]) Delete(k K) error {
	if !m.Contains(k) {
		return ErrValueNotExisted
	}
	delete(m.entries, k)
	return nil
}

func (m *hashMap[K, V]) Size() int {
	return len(m.entries)
}

func (m *hashMap[K, V]) Keys() []K {
	arr := make([]K, 0, m.Size())
	for k := range m.entries {
		arr = append(arr, k)
	}
	return arr
}

func (m *hashMap[K, V]) Values() []V {
	arr := make([]V, 0, m.Size())
	for _, v := range m.entries {
		arr = append(arr, v)
	}
	return arr
}
