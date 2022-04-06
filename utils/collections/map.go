package collections

type Map[K any, V any] interface {
	Contains(k K) bool
	Put(k K, v V, forced bool) error
	Get(k K) (V, error)
	Delete(k K) error
	Size() int
	Keys() []K
	Values() []V
}
