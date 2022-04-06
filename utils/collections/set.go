package collections

type Set[V any] interface {
	Contains(v V) bool
	Add(v V) error
	Remove(v V) error
	Size() int
	Entries() []V
}
