package math

import "math"

const MaxInt64 int64 = math.MaxInt64

func Min(a, b int64) int64 {
	if a < b {
		return a
	}

	return b
}

func Max(a, b int64) int64 {
	if a > b {
		return a
	}

	return b
}
