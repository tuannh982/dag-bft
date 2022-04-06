package collections

import "errors"

var (
	ErrValueExisted    = errors.New("value existed")
	ErrValueNotExisted = errors.New("value not existed")
)
