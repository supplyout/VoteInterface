package utils

import (
	"crypto/rand"
	"encoding/binary"
	"math/bits"
)

// RandUint64 return a random value between 0 and 0xFFFFFFFFFFFFFFFF
func RandUint64() uint64 {
	b := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return binary.BigEndian.Uint64(b)
}

// RandFloat64 returns a random float between min and max
func RandFloat64(min, max float64) float64 {
	b := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	f := float64(binary.BigEndian.Uint64(b)) / 1.8446744073709552e+19
	return min + f*(max-min)
}

// RandComplex128 returns a random complex with the real and imaginary part between min and max
func RandComplex128(min, max float64) complex128 {
	return complex(RandFloat64(min, max), RandFloat64(min, max))
}

// EqualSliceUint64 checks the equality between two uint64 slices.
func EqualSliceUint64(a, b []uint64) (v bool) {
	v = true
	for i := range a {
		v = v && (a[i] == b[i])
	}
	return
}

// EqualSliceInt64 checks the equality between two iint64 slices.
func EqualSliceInt64(a, b []int64) (v bool) {
	v = true
	for i := range a {
		v = v && (a[i] == b[i])
	}
	return
}

// EqualSliceUint8 checks the equality between two uint8 slices.
func EqualSliceUint8(a, b []uint8) (v bool) {
	v = true
	for i := range a {
		v = v && (a[i] == b[i])
	}
	return
}

// IsInSliceUint64 checks if x is in slice.
func IsInSliceUint64(x uint64, slice []uint64) (v bool) {
	for i := range slice {
		v = v || (slice[i] == x)
	}
	return
}

// MinUint64 returns the minimum value of the input slice of uint64 values.
func MinUint64(a, b uint64) (r uint64) {
	if a <= b {
		return a
	}
	return b
}

// MaxUint64 returns the maximum value of the input slice of uint64 values.
func MaxUint64(a, b uint64) (r uint64) {
	if a >= b {
		return a
	}
	return b
}

// MaxFloat64 returns the maximum value of the input slice of uint64 values.
func MaxFloat64(a, b float64) (r float64) {
	if a >= b {
		return a
	}
	return b
}

// BitReverse64 returns the bit-reverse value of the input value, within a context of 2^bitLen.
func BitReverse64(index, bitLen uint64) uint64 {
	return bits.Reverse64(index) >> (64 - bitLen)
}

// HammingWeight64 returns the hammingweight if the input value.
func HammingWeight64(x uint64) uint64 {
	x -= (x >> 1) & 0x5555555555555555
	x = (x & 0x3333333333333333) + ((x >> 2) & 0x3333333333333333)
	x = (x + (x >> 4)) & 0x0f0f0f0f0f0f0f0f
	return ((x * 0x0101010101010101) & 0xffffffffffffffff) >> 56
}

// AllDistinct returns true if all elements in s are distinct, and false otherwise.
func AllDistinct(s []uint64) bool {
	m := make(map[uint64]struct{}, len(s))
	for _, si := range s {
		if _, exists := m[si]; exists {
			return false
		}
		m[si] = struct{}{}
	}
	return true
}
