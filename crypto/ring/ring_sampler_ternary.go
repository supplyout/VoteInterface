package ring

import (
	"math"
	"math/bits"

	"github.com/ldsec/lattigo/v2/utils"
)

// TernarySampler keeps the state of a polynomial sampler in the ternary distribution.
type TernarySampler struct {
	baseSampler
	matrixProba  [2][precision - 1]uint8
	matrixValues [][3]uint64
	p            float64
	hw           uint64
	sample       func(lvl uint64, poly *Poly)
	samplemy     func(index []int, lvl uint64, pol *Poly)
}

// NewTernarySampler creates a new instance of TernarySampler from a PRNG, the ring definition and the distribution
// parameters: p is the probability of a coefficient being 0, (1-p)/2 is the probability of 1 and -1. If "montgomery"
// is set to true, polynomials read from this sampler are in Montgomery form.

func NewTernarySampler(prng utils.PRNG, baseRing *Ring, p float64, montgomery bool) *TernarySampler {
	ternarySampler := new(TernarySampler)
	ternarySampler.baseRing = baseRing
	ternarySampler.prng = prng
	ternarySampler.p = p
	//ternarySampler.sample = ternarySampler.sampleProba
	ternarySampler.sample = ternarySampler.sampleProba
	//修改的部分
	ternarySampler.samplemy = ternarySampler.sampleMy

	ternarySampler.initializeMatrix(montgomery)

	if p != 0.5 {
		ternarySampler.computeMatrixTernary(p)
	}

	return ternarySampler
}

// NewTernarySamplerSparse creates a new instance of a fixed-hamming-weight TernarySampler from a PRNG, the ring definition and the desired
// hamming weight for the output polynomials. If "montgomery" is set to true, polynomials read from this sampler
// are in Montgomery form.
func NewTernarySamplerSparse(prng utils.PRNG, baseRing *Ring, hw uint64, montgomery bool) *TernarySampler {
	ternarySampler := new(TernarySampler)
	ternarySampler.baseRing = baseRing
	ternarySampler.prng = prng
	ternarySampler.hw = hw
	ternarySampler.sample = ternarySampler.sampleSparse

	ternarySampler.initializeMatrix(montgomery)

	return ternarySampler
}

// Read samples a polynomial into pol.
func (ts *TernarySampler) Read(pol *Poly) {
	ts.sample(uint64(len(ts.baseRing.Modulus)-1), pol)
}

// ReadLvl samples a polynomial into pol at the speciefied level.
func (ts *TernarySampler) ReadLvl(lvl uint64, pol *Poly) {
	ts.sample(lvl, pol)
}

// ReadNew allocates and samples a polynomial at the max level.
func (ts *TernarySampler) ReadNew() (pol *Poly) {
	pol = ts.baseRing.NewPoly()
	ts.sample(uint64(len(ts.baseRing.Modulus)-1), pol)
	return pol
}

//按照index生成相对应的蒙哥马利表示的多项式
func (ts *TernarySampler) ReadNewMy(index []int) (pol *Poly) {
	pol = ts.baseRing.NewPoly()
	coeffs := make([][]uint64, len(pol.Coeffs))
	for i := 0; i < pol.GetLenModuli(); i++ {
		coeffs[i] = make([]uint64, len(pol.Coeffs[i]))
	}
	for i := 0; i < len(index); i++ {
		ans := ts.MFORM(index[i])
		for j := 0; j < pol.GetLenModuli(); j++ {
			coeffs[j][i] = ans[j]
		}
	}
	pol.SetCoefficients(coeffs)
	return pol
}

// ReadLvlNew allocates and samples a polynomial at the speficied level.
func (ts *TernarySampler) ReadLvlNew(lvl uint64) (pol *Poly) {
	pol = ts.baseRing.NewPolyLvl(lvl)
	ts.sample(lvl, pol)
	return pol
}

func (ts *TernarySampler) initializeMatrix(montgomery bool) {
	ts.matrixValues = make([][3]uint64, len(ts.baseRing.Modulus))

	// [0] = 0
	// [1] = 1 * 2^64 mod qi
	// [2] = (qi - 1) * 2^64 mod qi
	for i, Qi := range ts.baseRing.Modulus {
		ts.matrixValues[i][0] = 0

		if montgomery {
			ts.matrixValues[i][1] = MForm(1, Qi, ts.baseRing.BredParams[i])
			ts.matrixValues[i][2] = MForm(Qi-1, Qi, ts.baseRing.BredParams[i])
		} else {
			ts.matrixValues[i][1] = 1
			ts.matrixValues[i][2] = Qi - 1
		}
	}
}

// 将一个无符号整数转换成蒙哥马利表示形式，返回的数组表示该值在不同模数下的蒙哥马利表示形式
func (ts *TernarySampler) MFORM(x int) (ans []uint64) {
	//ts.matrixValues = make([][3]uint64, len(ts.baseRing.Modulus))
	ans = make([]uint64, len(ts.baseRing.Modulus))
	//println("MFORM函数", x)
	tmp := 0
	for i, Qi := range ts.baseRing.Modulus {
		//println(Qi)
		if x < 0 {
			tmp = int(Qi) + x
		} else {
			tmp = x
		}
		//println("tmp: ", tmp)
		ans[i] = MForm(uint64(tmp), Qi, ts.baseRing.BredParams[i])
	}
	//println(ans[0], " ", ans[1], " ", ans[2])
	return ans
}

func (ts *TernarySampler) computeMatrixTernary(p float64) {
	var g float64
	var x uint64

	g = p
	g *= math.Exp2(float64(precision))
	x = uint64(g)

	for j := uint64(0); j < precision-1; j++ {
		ts.matrixProba[0][j] = uint8((x >> (precision - j - 1)) & 1)
	}

	g = 1 - p
	g *= math.Exp2(float64(precision))
	x = uint64(g)

	for j := uint64(0); j < precision-1; j++ {
		ts.matrixProba[1][j] = uint8((x >> (precision - j - 1)) & 1)
	}

}

func (ts *TernarySampler) sampleProba(lvl uint64, pol *Poly) {

	if ts.p == 0 {
		panic("cannot sample -> p = 0")
	}

	var coeff uint64
	var sign uint64
	var index uint64

	if ts.p == 0.5 {

		randomBytesCoeffs := make([]byte, ts.baseRing.N>>3)
		randomBytesSign := make([]byte, ts.baseRing.N>>3)

		ts.prng.Clock(randomBytesCoeffs)

		ts.prng.Clock(randomBytesSign)

		for i := uint64(0); i < ts.baseRing.N; i++ {
			coeff = uint64(uint8(randomBytesCoeffs[i>>3])>>(i&7)) & 1
			sign = uint64(uint8(randomBytesSign[i>>3])>>(i&7)) & 1

			index = (coeff & (sign ^ 1)) | ((sign & coeff) << 1)

			for j := uint64(0); j < lvl+1; j++ {
				pol.Coeffs[j][i] = ts.matrixValues[j][index]
			}
		}

	} else {

		randomBytes := make([]byte, ts.baseRing.N)

		pointer := uint8(0)
		bytePointer := uint64(0)

		ts.prng.Clock(randomBytes)

		for i := uint64(0); i < ts.baseRing.N; i++ {

			coeff, sign, randomBytes, pointer, bytePointer = ts.kysampling(ts.prng, randomBytes, pointer, bytePointer, ts.baseRing.N)

			index = (coeff & (sign ^ 1)) | ((sign & coeff) << 1)

			for j := uint64(0); j < lvl+1; j++ {
				pol.Coeffs[j][i] = ts.matrixValues[j][index]
			}
		}
	}
}

//自己写的函数
func (ts *TernarySampler) sampleMy(index []int, lvl uint64, pol *Poly) {

	if ts.p == 0 {
		panic("cannot sample -> p = 0")
	}

	if ts.p == 0.5 {

		randomBytesCoeffs := make([]byte, ts.baseRing.N>>3)
		randomBytesSign := make([]byte, ts.baseRing.N>>3)

		ts.prng.Clock(randomBytesCoeffs)

		ts.prng.Clock(randomBytesSign)

		for i := uint64(0); i < ts.baseRing.N; i++ {

			for j := uint64(0); j < lvl+1; j++ {
				pol.Coeffs[j][i] = ts.matrixValues[j][index[i]]
			}
		}

	} else {

		randomBytes := make([]byte, ts.baseRing.N)

		ts.prng.Clock(randomBytes)

		for i := uint64(0); i < ts.baseRing.N; i++ {

			for j := uint64(0); j < lvl+1; j++ {
				pol.Coeffs[j][i] = ts.matrixValues[j][index[i]]
			}
		}
	}
}

func (ts *TernarySampler) sampleSparse(lvl uint64, pol *Poly) {

	if ts.hw > ts.baseRing.N {
		ts.hw = ts.baseRing.N
	}

	var mask, j uint64
	var coeff uint8

	index := make([]uint64, ts.baseRing.N)
	for i := uint64(0); i < ts.baseRing.N; i++ {
		index[i] = i
	}

	randomBytes := make([]byte, uint64(math.Ceil(float64(ts.hw)/8.0))) // We sample ceil(hw/8) bytes
	pointer := uint8(0)

	ts.prng.Clock(randomBytes)

	for i := uint64(0); i < ts.hw; i++ {
		mask = (1 << uint64(bits.Len64(ts.baseRing.N-i))) - 1 // rejection sampling of a random variable between [0, len(index)]

		j = randInt32(ts.prng, mask)
		for j >= ts.baseRing.N-i {
			j = randInt32(ts.prng, mask)
		}

		coeff = (uint8(randomBytes[0]) >> (i & 7)) & 1 // random binary digit [0, 1] from the random bytes (0 = 1, 1 = -1)
		for k := uint64(0); k < lvl+1; k++ {
			pol.Coeffs[k][index[j]] = ts.matrixValues[k][coeff+1]
		}

		// Remove the element in position j of the slice (order not preserved)
		index[j] = index[len(index)-1]
		index = index[:len(index)-1]

		pointer++

		if pointer == 8 {
			randomBytes = randomBytes[1:]
			pointer = 0
		}
	}
}

// kysampling uses the binary expansion and random bytes matrix to sample a discrete Gaussian value and its sign.
func (ts *TernarySampler) kysampling(prng utils.PRNG, randomBytes []byte, pointer uint8, bytePointer uint64,
	byteLength uint64) (uint64, uint64, []byte, uint8, uint64) {

	var sign uint8

	d := 0
	col := 0
	colLen := len(ts.matrixProba)

	for {

		// Use one random byte per cycle and cycle through the randomBytes
		for i := pointer; i < 8; i++ {

			d = (d << 1) + 1 - int((uint8(randomBytes[bytePointer])>>i)&1)

			// There is small probability that it will get out of the bound, then
			// rerun until it gets a proper output
			if d > colLen-1 {
				return ts.kysampling(prng, randomBytes, i, bytePointer, byteLength)
			}

			for row := colLen - 1; row >= 0; row-- {

				d -= int(ts.matrixProba[row][col])

				if d == -1 {

					// Sign
					if i == 7 {
						pointer = 0
						// If the last bit of the array was read, sample a new one
						bytePointer++

						if bytePointer >= byteLength {
							bytePointer = 0
							prng.Clock(randomBytes)
						}

						sign = uint8(randomBytes[bytePointer]) & 1

					} else {
						pointer = i
						// Otherwise, the sign is the next bit of the byte
						sign = uint8(randomBytes[bytePointer]>>(i+1)) & 1
					}

					return uint64(row), uint64(sign), randomBytes, pointer + 1, bytePointer
				}
			}

			col++
		}

		// Reset the bit pointer and discard the used byte
		pointer = 0
		// If the last bit of the array was read, sample a new one
		bytePointer++

		if bytePointer >= byteLength {
			bytePointer = 0
			prng.Clock(randomBytes)
		}

	}
}
