package math

import "golang.org/x/exp/constraints"

func DivCeil[T constraints.Integer](dividend, divisor T) T {
	base := dividend / divisor
	if dividend%divisor == 0 {
		return base
	} else {
		return base + 1
	}
}

func DivFloor[T constraints.Integer](dividend, divisor T) T {
	base := dividend / divisor
	return base
}
