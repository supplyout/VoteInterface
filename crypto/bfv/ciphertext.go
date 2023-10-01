package bfv

import "Vote/crypto/utils"

// Ciphertext is a *ring.Poly array representing a polynomial of degree > 0 with coefficients in R_Q.
type Ciphertext struct {
	*Element // 一个已经定义的结构体，表示一个多项式数组
}

// NewCiphertext creates a new ciphertext parameterized by degree, level and scale.
func NewCiphertext(params *Parameters, degree uint64) (ciphertext *Ciphertext) {
	return &Ciphertext{newCiphertextElement(params, degree)}
}

// NewCiphertextRandom generates a new uniformly distributed ciphertext of degree, level and scale.
func NewCiphertextRandom(prng utils.PRNG, params *Parameters, degree uint64) (ciphertext *Ciphertext) {
	ciphertext = &Ciphertext{newCiphertextElement(params, degree)}
	populateElementRandom(prng, params, ciphertext.Element)
	return
}
