package bfv

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/bits"

	//"github.com/ldsec/lattigo/v2/ring"
	//"github.com/ldsec/lattigo/v2/utils"
	"Vote/crypto/ring"
	"Vote/crypto/utils"
)

// MaxLogN is the log2 of the largest supported polynomial modulus degree.
const MaxLogN = 16

// MinLogN is the log2 of the smallest supported polynomial modulus degree (用来确保NTT的正确性).
const MinLogN = 4

// MaxModuliCount is the largest supported number of moduli in the RNS representation.
const MaxModuliCount = 34

// MaxModuliSize is the largest bit-length supported for the moduli in the RNS representation.
const MaxModuliSize = 60

const (
	// PN12QP109 is a set of parameters with N = 2^12 and log(QP) = 109
	// 特殊常量，可以认为是一个可以被编译器修改的常量
	//iota在const关键字出现时将被重置为 0，
	//const中每新增一行常量声明将使iota计数一次
	PN12QP109 = iota // 0
	// PN13QP218 is a set of parameters with N = 2^13 and log(QP) = 218
	PN13QP218 // 1
	// PN14QP438 is a set of parameters with N = 2^14 and log(QP) = 438
	PN14QP438 // 2
	// PN15QP880 is a set of parameters with N = 2^15 and log(QP) = 880
	PN15QP880

	// PN12QP101pq is the index in DefaultParams for logQP = 101 (post quantum)
	PN12QP101pq
	// PN13QP202pq is the index in DefaultParams for logQP = 202 (post quantum)
	PN13QP202pq
	// PN14QP411pq is the index in DefaultParams for logQP = 411 (post quantum)
	PN14QP411pq
	// PN15QP827pq is the index in DefaultParams for logQP = 827 (post quantum)
	PN15QP827pq
)

// DefaultSigma is the default error distribution standard deviation
const DefaultSigma = 3.2

// DefaultParams is a set of default BFV parameters ensuring 128 bit security.
var DefaultParams = []*Parameters{

	{
		logN:  12,
		t:     65537,
		qi:    []uint64{0x7ffffec001, 0x8000016001}, // 39 + 39 bits
		pi:    []uint64{0x40002001},                 // 30 bits
		sigma: DefaultSigma,
	},

	{
		logN:  13,
		t:     65537,
		qi:    []uint64{0x3fffffffef8001, 0x4000000011c001, 0x40000000120001}, // 54 + 54 + 54 bits
		pi:    []uint64{0x7ffffffffb4001},                                     // 55 bits
		sigma: DefaultSigma,
	},

	{
		logN: 14,
		t:    65537,
		qi: []uint64{0x100000000060001, 0x80000000068001, 0x80000000080001,
			0x3fffffffef8001, 0x40000000120001, 0x3fffffffeb8001}, // 56 + 55 + 55 + 54 + 54 + 54 bits
		pi:    []uint64{0x80000000130001, 0x7fffffffe90001}, // 55 + 55 bits
		sigma: DefaultSigma,
	},

	{
		logN: 15,
		t:    65537,
		qi: []uint64{0x7ffffffffe70001, 0x7ffffffffe10001, 0x7ffffffffcc0001, // 59 + 59 + 59 bits
			0x400000000270001, 0x400000000350001, 0x400000000360001, // 58 + 58 + 58 bits
			0x3ffffffffc10001, 0x3ffffffffbe0001, 0x3ffffffffbd0001, // 58 + 58 + 58 bits
			0x4000000004d0001, 0x400000000570001, 0x400000000660001}, // 58 + 58 + 58 bits
		pi:    []uint64{0xffffffffffc0001, 0x10000000001d0001, 0x10000000006e0001}, // 60 + 60 + 60 bits
		sigma: DefaultSigma,
	},

	{ // LogQP = 101.00005709794536
		logN:  12,
		t:     65537,
		qi:    []uint64{0x800004001, 0x800008001}, // 2*35
		pi:    []uint64{0x80014001},               // 1*31
		sigma: DefaultSigma,
	},

	{ // LogQP = 201.99999999994753
		logN:  13,
		t:     65537,
		qi:    []uint64{0x7fffffffe0001, 0x7fffffffcc001, 0x3ffffffffc001}, // 2*51 + 50
		pi:    []uint64{0x4000000024001},                                   // 50
		sigma: DefaultSigma,
	},

	{ // LogQP = 410.9999999999886
		logN:  14,
		t:     65537,
		qi:    []uint64{0x7fffffffff18001, 0x8000000000f8001, 0x7ffffffffeb8001, 0x800000000158001, 0x7ffffffffe70001}, // 5*59
		pi:    []uint64{0x7ffffffffe10001, 0x400000000068001},                                                          // 59+58
		sigma: DefaultSigma,
	},

	{ // LogQP = 826.9999999999509
		logN: 15,
		t:    65537,
		qi: []uint64{0x7ffffffffe70001, 0x7ffffffffe10001, 0x7ffffffffcc0001, 0x7ffffffffba0001, 0x8000000004a0001,
			0x7ffffffffb00001, 0x800000000890001, 0x8000000009d0001, 0x7ffffffff630001, 0x800000000a70001,
			0x7ffffffff510001}, // 11*59
		pi:    []uint64{0x800000000b80001, 0x800000000bb0001, 0xffffffffffc0001}, // 2*59+60
		sigma: DefaultSigma,
	},
}

// Moduli stores the NTT primes of the RNS representation.
type Moduli struct {
	Qi []uint64 // Ciphertext prime moduli
	Pi []uint64 // Keys additional prime moduli
}

// Copy creates a copy of the target Moduli.
func (m *Moduli) Copy() Moduli {

	qi := make([]uint64, len(m.Qi))
	copy(qi, m.Qi)

	pi := make([]uint64, len(m.Pi))
	copy(pi, m.Pi)

	return Moduli{qi, pi}
}

// LogModuli stores the bit-length of the NTT primes of the RNS representation.
type LogModuli struct {
	LogQi []uint64 // Ciphertext prime moduli bit-size
	LogPi []uint64 // Keys additional prime moduli bit-size
}

// Copy creates a copy of the target LogModuli.
func (m *LogModuli) Copy() LogModuli {

	LogQi := make([]uint64, len(m.LogQi))
	copy(LogQi, m.LogQi)

	LogPi := make([]uint64, len(m.LogPi))
	copy(LogPi, m.LogPi)

	return LogModuli{LogQi, LogPi}
}

// Parameters represents a given parameter set for the BFV cryptosystem.
type Parameters struct {
	logN  uint64 // Log Ring degree (power of 2)
	qi    []uint64
	pi    []uint64
	t     uint64  // Plaintext modulus
	sigma float64 // Gaussian sampling standard deviation
}

// NewParametersFromModuli creates a new Parameters struct and returns a pointer to it.
func NewParametersFromModuli(logN uint64, m *Moduli, t uint64) (p *Parameters, err error) {

	p = new(Parameters)

	if logN < MinLogN || logN > MaxLogN {
		return nil, fmt.Errorf("invalid polynomial ring log degree: %d", logN)
	}

	p.logN = logN

	// Checks if Moduli is valid
	if err = checkModuli(m, logN); err != nil {
		return nil, err
	}

	p.qi = make([]uint64, len(m.Qi))
	copy(p.qi, m.Qi)

	p.pi = make([]uint64, len(m.Pi))
	copy(p.pi, m.Pi)

	p.sigma = DefaultSigma

	p.t = t

	return p, nil
}

// NewParametersFromLogModuli creates a new Parameters struct and returns a pointer to it.
func NewParametersFromLogModuli(logN uint64, lm *LogModuli, t uint64) (p *Parameters, err error) {

	if err = checkLogModuli(lm); err != nil {
		return nil, err
	}

	// If LogModuli is valid and then generates the moduli
	return NewParametersFromModuli(logN, genModuli(lm, logN), t)
}

// LogN returns the log of the degree of the polynomial ring
func (p *Parameters) LogN() uint64 {
	return p.logN
}

// N returns power of two degree of the ring
func (p *Parameters) N() uint64 {
	return 1 << p.logN
}

// T returns the plaintext coefficient modulus t
func (p *Parameters) T() uint64 {
	return p.t
}

// Sigma returns standard deviation of the noise distribution
func (p *Parameters) Sigma() float64 {
	return p.sigma
}

// SetT sets the plaintext coefficient modulus t
func (p *Parameters) SetT(T uint64) {
	p.t = T
}

// WithT returns a copy of the parmaters with a plaintext modulus set to T
// 参数是明文的模量T，返回一组新的参数
func (p *Parameters) WithT(T uint64) (pCopy *Parameters) {
	pCopy = p.Copy()
	pCopy.SetT(T)
	return
}

// LogModuli generates a LogModuli struct from the parameters' Moduli struct and returns it.
func (p *Parameters) LogModuli() (lm *LogModuli) {
	lm = new(LogModuli)
	lm.LogQi = make([]uint64, len(p.qi))
	for i := range p.qi {
		lm.LogQi[i] = uint64(math.Round(math.Log2(float64(p.qi[i]))))
	}
	lm.LogPi = make([]uint64, len(p.pi))
	for i := range p.pi {
		lm.LogPi[i] = uint64(math.Round(math.Log2(float64(p.pi[i]))))
	}
	return
}

// Moduli returns a struct Moduli with the moduli of the parameters
//返回的是m，也就是返回了参数表示地参数的所有的p和q
func (p *Parameters) Moduli() (m *Moduli) {
	m = new(Moduli)
	m.Qi = make([]uint64, len(p.qi))
	copy(m.Qi, p.qi)
	m.Pi = make([]uint64, len(p.pi))
	copy(m.Pi, p.pi)

	return
}

// Qi returns a new slice with the factors of the ciphertext modulus q
// 返回一个新的切片，其中包含密文模量q的因子
func (p *Parameters) Qi() []uint64 {
	qi := make([]uint64, len(p.qi))
	copy(qi, p.qi)
	return qi
}

// QiCount returns the number of factors of the ciphertext modulus q
func (p *Parameters) QiCount() uint64 {
	return uint64(len(p.qi))
}

// Pi returns a new slice with the factors of the ciphertext modulus extension P
func (p *Parameters) Pi() []uint64 {
	pi := make([]uint64, len(p.pi))
	copy(pi, p.pi)
	return pi
}

// PiCount returns the number of factors of the ciphertext modulus extension P
// 密文模量扩展P的因子个数
func (p *Parameters) PiCount() uint64 {
	return uint64(len(p.pi))
}

// QPiCount returns the number of factors of the ciphertext modulus Q + the modulus extension P
func (p *Parameters) QPiCount() uint64 {
	return p.QiCount() + p.PiCount()
}

// LogQP returns the size of the extended modulus QP in bits
// 返回扩展模量QP的大小(以比特为单位)
func (p *Parameters) LogQP() uint64 {
	tmp := ring.NewUint(1)
	for _, qi := range p.qi {
		tmp.Mul(tmp, ring.NewUint(qi))
	}
	for _, pi := range p.pi {
		tmp.Mul(tmp, ring.NewUint(pi))
	}
	return uint64(tmp.BitLen())
}

// 返回QP的乘积
func (p *Parameters) QP() *big.Int {
	tmp := ring.NewUint(1)
	for _, qi := range p.qi {
		tmp.Mul(tmp, ring.NewUint(qi))
	}
	for _, pi := range p.pi {
		tmp.Mul(tmp, ring.NewUint(pi))
	}
	return tmp
}

// LogQ returns the size of the modulus Q in bits
// 返回模数Q的大小(以比特为单位)
func (p *Parameters) LogQ() uint64 {
	tmp := ring.NewUint(1)
	for _, qi := range p.qi {
		tmp.Mul(tmp, ring.NewUint(qi))
	}
	return uint64(tmp.BitLen())
}

// LogP returns the size of the modulus P in bits
// 返回模数P的大小(以比特为单位)
func (p *Parameters) LogP() uint64 {
	tmp := ring.NewUint(1)
	for _, pi := range p.pi {
		tmp.Mul(tmp, ring.NewUint(pi))
	}
	return uint64(tmp.BitLen())
}

// LogQAlpha returns the size in bits of the sum of the norm of
// each element of the special RNS decomposition basis for the
// key-switching.
// LogQAlpha 返回按键切换的特殊RNS分解基的每个元素的范数之和的大小(以比特为单位)。
// LogQAlpha is the size of the element that is multiplied by the
// error during the keyswitching and then divided by P.
// LogQAlpha 元素的大小乘以按键切换时的误差，然后除以P。
// LogQAlpha should be smaller than P or the error added during
// the key-switching wont be negligible.
// LogQAlpha 应小于P，否则按键切换过程中所加的误差不可忽略。
func (p *Parameters) LogQAlpha() uint64 {

	alpha := p.PiCount()

	if alpha == 0 {
		return 0
	}

	res := ring.NewUint(0)
	var j uint64
	for i := uint64(0); i < p.QiCount(); i = i + alpha {

		j = i + alpha
		if j > p.QiCount() {
			j = p.QiCount()
		}

		tmp := ring.NewUint(1)
		for _, qi := range p.qi[i:j] {
			tmp.Mul(tmp, ring.NewUint(qi))
		}

		res.Add(res, tmp)
	}

	return uint64(res.BitLen())
}

// Alpha returns the number of moduli in P
func (p *Parameters) Alpha() uint64 {
	return p.PiCount()
}

// Beta returns the number of element in the RNS decomposition basis: Ceil(lenQi / lenPi)
func (p *Parameters) Beta() uint64 {
	if p.Alpha() != 0 {
		return uint64(math.Ceil(float64(p.QiCount()) / float64(p.Alpha())))
	}

	return 0
}

// NewPolyQ returns a new empty polynomial of degree 2^logN in basis qi.
func (p *Parameters) NewPolyQ() *ring.Poly {
	return ring.NewPoly(p.N(), p.QiCount())
}

// NewPolyP returns a new empty polynomial of degree 2^logN in basis Pi.
func (p *Parameters) NewPolyP() *ring.Poly {
	return ring.NewPoly(p.N(), p.PiCount())
}

// NewPolyQP returns a new empty polynomial of degree 2^logN in basis qi + Pi.
func (p *Parameters) NewPolyQP() *ring.Poly {
	return ring.NewPoly(p.N(), p.QPiCount())
}

// Copy creates a copy of the target Parameters.
func (p *Parameters) Copy() (paramsCopy *Parameters) {

	paramsCopy = new(Parameters)
	paramsCopy.logN = p.logN
	paramsCopy.t = p.t
	paramsCopy.sigma = p.sigma
	paramsCopy.qi = make([]uint64, len(p.qi))
	copy(paramsCopy.qi, p.qi)
	paramsCopy.pi = make([]uint64, len(p.pi))
	copy(paramsCopy.pi, p.pi)
	return
}

// Equals compares two sets of parameters for equality.
func (p *Parameters) Equals(other *Parameters) (res bool) {

	if p == other {
		return true
	}

	res = p.logN == other.logN
	res = res && (p.t == other.t)
	res = res && (p.sigma == other.sigma)

	res = res && utils.EqualSliceUint64(p.qi, other.qi)
	res = res && utils.EqualSliceUint64(p.pi, other.pi)

	return
}

// MarshalBinary returns a []byte representation of the parameter set.
// 将所有的参数都转换成二进制形式返回
func (p *Parameters) MarshalBinary() ([]byte, error) {
	if p.logN == 0 { // if N is 0, then p is the zero value
		return []byte{}, nil
	}

	// data : 19 byte + len(QPi) * 8 byte
	// 1 byte : logN
	// 1 byte : #pi
	// 1 byte : #pi
	// 8 byte : t
	// 8 byte : sigma
	b := utils.NewBuffer(make([]byte, 0, 19+(len(p.qi)+len(p.pi))<<3))

	b.WriteUint8(uint8(p.logN))
	b.WriteUint8(uint8(len(p.qi)))
	b.WriteUint8(uint8(len(p.pi)))
	b.WriteUint64(p.t)
	b.WriteUint64(math.Float64bits(p.sigma))
	b.WriteUint64Slice(p.qi)
	b.WriteUint64Slice(p.pi)

	return b.Bytes(), nil
}

// UnmarshalBinary decodes a []byte into a parameter set struct.
// 将字节顺序的参数重新返回到正常格式
func (p *Parameters) UnmarshalBinary(data []byte) error {
	if len(data) < 19 {
		return errors.New("invalid parameters encoding")
	}
	b := utils.NewBuffer(data)

	p.logN = uint64(b.ReadUint8())

	if p.logN > MaxLogN {
		return fmt.Errorf("logN larger than %d", MaxLogN)
	}

	lenQi := b.ReadUint8()
	lenPi := b.ReadUint8()

	p.t = b.ReadUint64()
	p.sigma = math.Float64frombits(b.ReadUint64())
	p.qi = make([]uint64, lenQi)
	p.pi = make([]uint64, lenPi)

	b.ReadUint64Slice(p.qi)
	b.ReadUint64Slice(p.pi)

	err := checkModuli(p.Moduli(), p.logN)
	if err != nil {
		return err
	}
	return nil
}

func checkModuli(m *Moduli, logN uint64) (err error) {

	if len(m.Qi) > MaxModuliCount {
		return fmt.Errorf("#qi is larger than %d", MaxModuliCount)
	}

	if len(m.Pi) > MaxModuliCount {
		return fmt.Errorf("#Pi is larger than %d", MaxModuliCount)
	}

	for i, qi := range m.Qi {
		if uint64(bits.Len64(qi)-1) > MaxModuliSize+1 {
			return fmt.Errorf("qi bit-size for i=%d is larger than %d", i, MaxModuliSize)
		}
	}

	for i, pi := range m.Pi {
		if uint64(bits.Len64(pi)-1) > MaxModuliSize+1 {
			return fmt.Errorf("Pi bit-size for i=%d is larger than %d", i, MaxModuliSize)
		}
	}

	N := uint64(1 << logN)

	for i, qi := range m.Qi {
		if !ring.IsPrime(qi) || qi&((N<<1)-1) != 1 {
			return fmt.Errorf("qi n°%d is not an NTT prime", i)
		}
	}

	for i, pi := range m.Pi {
		if !ring.IsPrime(pi) || pi&((N<<1)-1) != 1 {
			return fmt.Errorf("Pi n°%d is not an NTT prime", i)
		}
	}

	return nil
}

func checkLogModuli(lm *LogModuli) (err error) {

	// Checks if the parameters are empty
	if lm.LogQi == nil || len(lm.LogQi) == 0 {
		return fmt.Errorf("nil or empty slice provided as LogModuli.LogQi")
	}

	if len(lm.LogQi) > MaxModuliCount {
		return fmt.Errorf("#LogQi is larger than %d", MaxModuliCount)
	}

	if len(lm.LogPi) > MaxModuliCount {
		return fmt.Errorf("#LogPi is larger than %d", MaxModuliCount)
	}

	for i, qi := range lm.LogQi {
		if qi > MaxModuliSize {
			return fmt.Errorf("LogQi for i=%d is larger than %d", i, MaxModuliSize)
		}
	}

	for i, pi := range lm.LogPi {
		if pi > MaxModuliSize {
			return fmt.Errorf("LogPi for i=%d is larger than %d", i, MaxModuliSize)
		}
	}

	return nil
}

// GenModuli generates the appropriate primes from the parameters using generateNTTPrimes
// such that all primes are different.
func genModuli(lm *LogModuli, logN uint64) (m *Moduli) {

	m = new(Moduli)

	// Extracts all the different primes bit-size and maps their number
	primesbitlen := make(map[uint64]uint64)

	for _, qi := range lm.LogQi {
		primesbitlen[qi]++
	}

	for _, pj := range lm.LogPi {
		primesbitlen[pj]++
	}

	// For each bit-size, it finds that many primes
	primes := make(map[uint64][]uint64)
	for key, value := range primesbitlen {
		primes[key] = ring.GenerateNTTPrimes(key, 2<<logN, value)
	}

	// Assigns the primes to the CKKS moduli chain
	m.Qi = make([]uint64, len(lm.LogQi))
	for i, qi := range lm.LogQi {
		m.Qi[i] = primes[qi][0]
		primes[qi] = primes[qi][1:]
	}

	// Assigns the primes to the special primes list for the extended ring
	m.Pi = make([]uint64, len(lm.LogPi))
	for i, pj := range lm.LogPi {
		m.Pi[i] = primes[pj][0]
		primes[pj] = primes[pj][1:]
	}

	return
}
