package bfv

import (
	"math/big"

	//"github.com/ldsec/lattigo/v2/ring"
	//"github.com/ldsec/lattigo/v2/utils"
	"Vote/crypto/ring"
	"Vote/crypto/utils"
)

// KeyGenerator is an interface implementing the methods of the keyGenerator.
type KeyGenerator interface {
	GenSecretKey() (sk *SecretKey)
	GenSecretkeyWithDistrib(p float64) (sk *SecretKey)
	GenPublicKey(sk *SecretKey) (pk *PublicKey)
	GenKeyPair() (sk *SecretKey, pk *PublicKey)
	GenRelinKey(sk *SecretKey, maxDegree uint64) (evk *EvaluationKey)
	GenSwitchingKey(skIn, skOut *SecretKey) (evk *SwitchingKey)
	GenRot(rotType Rotation, sk *SecretKey, k uint64, rotKey *RotationKeys)
	GenRotationKeysPow2(sk *SecretKey) (rotKey *RotationKeys)

	GenPublicKeyMy(sk *SecretKey, a, e1, e2 *ring.Poly) (pk *PublicKey)
	Restore(poly *ring.Poly)
	Trans(poly *ring.Poly)
	GenSecretKeyMy(coeffs []int, p float64) (sk *SecretKey)
	RestoreSki(ski []uint64) (sk *SecretKey)
	PolyFromArray(coeffs []int) *ring.Poly
	Mul(poly1, poly2 *ring.Poly) *ring.Poly
	Add(poly1, poly2 *ring.Poly) *ring.Poly
	Inv(poly *ring.Poly)
	PolyMFromArray(coeffs []int) *ring.Poly
	InvMFORM(poly *ring.Poly)
	NewSecretKeyFromMForm(coeffs [][]uint64) (sk *SecretKey)
	NewPolyFromCoeffs(coeffs []uint64) *ring.Poly
}

// keyGenerator is a structure that stores the elements required to create new keys,
// as well as a small memory pool for intermediate values.
type keyGenerator struct {
	params          *Parameters
	ringQP          *ring.Ring
	pBigInt         *big.Int
	polypool        [2]*ring.Poly
	gaussianSampler *ring.GaussianSampler
	uniformSampler  *ring.UniformSampler
}

// SecretKey is a structure that stores the SecretKey.
type SecretKey struct {
	sk *ring.Poly
}

// PublicKey is a structure that stores the PublicKey.
type PublicKey struct {
	pk [2]*ring.Poly
}

// Rotation is a type used to represent the rotations types.
type Rotation int

// Constants for rotation types
const (
	RotationRight = iota + 1
	RotationLeft
	RotationRow
)

// RotationKeys is a structure that stores the switching-keys required during the homomorphic rotations.
type RotationKeys struct {
	evakeyRotColLeft  map[uint64]*SwitchingKey
	evakeyRotColRight map[uint64]*SwitchingKey
	evakeyRotRow      *SwitchingKey
}

// EvaluationKey is a structure that stores the switching-keys required during the relinearization.
type EvaluationKey struct {
	evakey []*SwitchingKey
}

// SwitchingKey is a structure that stores the switching-keys required during the key-switching.
type SwitchingKey struct {
	evakey [][2]*ring.Poly
}

// Get returns the switching key backing slice.
func (swk *SwitchingKey) Get() [][2]*ring.Poly {
	return swk.evakey
}

// NewKeyGenerator creates a new KeyGenerator,
// from which the secret and public keys, as well as the evaluation,
// rotation and switching keys can be generated.
func NewKeyGenerator(params *Parameters) KeyGenerator {

	var ringQP *ring.Ring
	var err error
	if ringQP, err = ring.NewRing(params.N(), append(params.qi, params.pi...)); err != nil {
		panic(err)
	}

	var pBigInt *big.Int
	if len(params.pi) != 0 {
		pBigInt = ring.NewUint(1)
		for _, pi := range params.pi {
			pBigInt.Mul(pBigInt, ring.NewUint(pi))
		}
	}

	prng, err := utils.NewPRNG()
	if err != nil {
		panic(err)
	}

	return &keyGenerator{
		params:          params.Copy(),
		ringQP:          ringQP,
		pBigInt:         pBigInt,
		polypool:        [2]*ring.Poly{ringQP.NewPoly(), ringQP.NewPoly()},
		gaussianSampler: ring.NewGaussianSampler(prng, ringQP, params.Sigma(), uint64(6*params.Sigma())),
		uniformSampler:  ring.NewUniformSampler(prng, ringQP),
	}
}

// GenSecretKey creates a new SecretKey with the distribution [1/3, 1/3, 1/3].
func (keygen *keyGenerator) GenSecretKey() (sk *SecretKey) {
	return keygen.GenSecretkeyWithDistrib(1.0 / 3)
}

// 将参与方的蒙哥马利表示格式的ski数组转换成NTT格式表示的私钥

func (keygen *keyGenerator) RestoreSki(ski []uint64) (sk *SecretKey) {
	sk = new(SecretKey)
	sk.sk = keygen.ringQP.NewPoly()
	tmp_array := make([]*uint64, keygen.ringQP.N)
	keygen.ringQP.InvMFormArray(ski, tmp_array) // 将蒙哥马利表示形式的数组转换成最朴素形式
	//println("还原ski")
	println(tmp_array[0])
	keygen.ringQP.MFormArrayUint(tmp_array, sk.sk) // 生成蒙哥马利表示形式的多项式
	println("蒙哥马利表示形式的sk")
	keygen.ringQP.InvMForm(sk.sk, sk.sk)

	for i := 0; i < 16; i++ {
		println(sk.sk.Coeffs[0][i], " ", sk.sk.Coeffs[1][i], " ", sk.sk.Coeffs[2][i])
	}

	keygen.ringQP.MForm(sk.sk, sk.sk)
	keygen.ringQP.NTT(sk.sk, sk.sk) // 将蒙哥马利表示形式的多项式转换成NTT格式
	return sk
}

// 将一个int类型的数组转换成一个NTT格式的多项式
func (keygen *keyGenerator) PolyFromArray(coeffs []int) *ring.Poly {
	poly := keygen.ringQP.NewPoly()
	keygen.ringQP.MFormArrayInt(coeffs, poly)
	keygen.ringQP.NTT(poly, poly)
	return poly
}

// 将一个蒙哥马利表示形式的多项式转换成NTT格式的多项式
func (keygen *keyGenerator) PolyMFromArray(coeffs []int) *ring.Poly {
	poly := keygen.ringQP.NewPoly()
	keygen.ringQP.MFormArrayInt(coeffs, poly)
	keygen.ringQP.NTT(poly, poly)
	return poly
}

func (keygen *keyGenerator) Mul(poly1, poly2 *ring.Poly) *ring.Poly {
	poly := keygen.ringQP.NewPoly()
	keygen.ringQP.MulCoeffsMontgomery(poly1, poly2, poly)
	return poly
}

func (keygen *keyGenerator) Add(poly1, poly2 *ring.Poly) *ring.Poly {
	poly := keygen.ringQP.NewPoly()
	keygen.ringQP.Add(poly1, poly2, poly)
	return poly
}

func (keygen *keyGenerator) Inv(poly *ring.Poly) {
	keygen.ringQP.InvNTT(poly, poly)
	keygen.ringQP.InvMForm(poly, poly)
}

func (keygen *keyGenerator) InvMFORM(poly *ring.Poly) {
	//keygen.ringQP.InvNTT(poly, poly)
	keygen.ringQP.InvMForm(poly, poly)
}

// 将coeffs表示的系数转换成蒙哥马利形式，进而转换成NTT格式，作为密钥进行输出
func (keygen *keyGenerator) GenSecretKeyMy(coeffs []int, p float64) (sk *SecretKey) {

	prng, err := utils.NewPRNG()
	if err != nil {
		panic(err)
	}
	ternarySamplerMontgomery := ring.NewTernarySampler(prng, keygen.ringQP, p, true)
	sk = new(SecretKey)
	sk.sk = ternarySamplerMontgomery.ReadNewMy(coeffs) // 蒙哥马利域表示形式
	/*
		println("蒙哥马利表示形式的sk")
		keygen.ringQP.InvMForm(sk.sk, sk.sk)

		for i := 0; i < 16; i++{
			println(sk.sk.Coeffs[0][i])
		}

		keygen.ringQP.MForm(sk.sk, sk.sk)

	*/
	keygen.ringQP.NTT(sk.sk, sk.sk)
	return sk
}

// GenSecretkeyWithDistrib creates a new SecretKey with the distribution [(1-p)/2, p, (1-p)/2].
func (keygen *keyGenerator) GenSecretkeyWithDistrib(p float64) (sk *SecretKey) {
	prng, err := utils.NewPRNG()
	if err != nil {
		panic(err)
	}
	ternarySamplerMontgomery := ring.NewTernarySampler(prng, keygen.ringQP, p, true)

	sk = new(SecretKey)
	sk.sk = ternarySamplerMontgomery.ReadNew() // 蒙哥马利域表示形式

	keygen.ringQP.NTT(sk.sk, sk.sk)
	return sk
}

// NewSecretKey generates a new SecretKey with zero values.
func NewSecretKey(params *Parameters) *SecretKey {

	sk := new(SecretKey)
	sk.sk = ring.NewPoly(uint64(1<<params.logN), uint64(len(params.qi)+len(params.pi)))
	return sk
}

// 利用一个蒙哥马利表示形式的数组生成一个私钥sk
func (keygen *keyGenerator) NewSecretKeyFromMForm(coeffs [][]uint64) (sk *SecretKey) {
	sk = new(SecretKey)
	sk.sk = keygen.ringQP.NewPoly()
	sk.sk.SetCoefficients(coeffs)
	keygen.ringQP.NTT(sk.sk, sk.sk)
	return
}

func (keygen *keyGenerator) NewPolyFromCoeffs(coeffs []uint64) *ring.Poly {
	poly := keygen.ringQP.NewPoly()
	keygen.ringQP.MFormArray(coeffs, poly)
	keygen.ringQP.NTT(poly, poly)
	return poly
}

// Get returns the polynomial of the target SecretKey.
func (sk *SecretKey) Get() *ring.Poly {
	return sk.sk
}

// Set sets the polynomial of the target secret key as the input polynomial.
func (sk *SecretKey) Set(poly *ring.Poly) {
	sk.sk = poly.CopyNew()
}

// GenPublicKey generates a new PublicKey from the provided SecretKey.
func (keygen *keyGenerator) GenPublicKey(sk *SecretKey) (pk *PublicKey) {

	pk = new(PublicKey)

	ringQP := keygen.ringQP

	//pk[0] = [-(a*s + e)]
	//pk[1] = [a]

	pk.pk[0] = keygen.gaussianSampler.ReadNew()
	ringQP.NTT(pk.pk[0], pk.pk[0])
	pk.pk[1] = keygen.uniformSampler.ReadNew()

	ringQP.MulCoeffsMontgomeryAndSub(sk.sk, pk.pk[1], pk.pk[0])

	return pk
}
func (keygen *keyGenerator) Restore(poly *ring.Poly) {
	keygen.ringQP.InvNTT(poly, poly)
	keygen.ringQP.InvMForm(poly, poly)
}
func (keygen *keyGenerator) Trans(poly *ring.Poly) {
	keygen.ringQP.MForm(poly, poly)
	keygen.ringQP.NTT(poly, poly)
	//keygen.ringQP.InvMForm(poly, poly)
}

// GenPublicKey generates a new PublicKey from the provided SecretKey.
// 修改过的生成pk的函数
func (keygen *keyGenerator) GenPublicKeyMy(sk *SecretKey, a, e1, e2 *ring.Poly) (pk *PublicKey) {

	pk = new(PublicKey)

	ringQP := keygen.ringQP

	//pk[0] = [-(a*s + e)]
	//pk[1] = [a]
	pk.pk[0] = ring.NewPoly(keygen.params.N(), keygen.params.QPiCount())
	ringQP.Add(e1, e2, pk.pk[0])
	//ringQP.NTT(pk.pk[0], pk.pk[0])
	pk.pk[1] = ring.NewPoly(keygen.params.N(), keygen.params.QPiCount())
	pk.pk[1] = a

	ringQP.MulCoeffsMontgomeryAndSub(sk.sk, pk.pk[1], pk.pk[0])

	return pk
}

// NewPublicKey returns a new PublicKey with zero values.
func NewPublicKey(params *Parameters) (pk *PublicKey) {

	pk = new(PublicKey)

	pk.pk[0] = ring.NewPoly(uint64(1<<params.logN), uint64(len(params.qi)+len(params.pi)))
	pk.pk[1] = ring.NewPoly(uint64(1<<params.logN), uint64(len(params.qi)+len(params.pi)))

	return
}

// Get returns the polynomials of the PublicKey.
func (pk *PublicKey) Get() [2]*ring.Poly {
	return pk.pk
}

// Set sets the polynomial of the PublicKey as the input polynomials.
func (pk *PublicKey) Set(p [2]*ring.Poly) {
	pk.pk[0] = p[0].CopyNew()
	pk.pk[1] = p[1].CopyNew()
}

// NewKeyPair generates a new SecretKey with distribution [1/3, 1/3, 1/3] and a corresponding PublicKey.
func (keygen *keyGenerator) GenKeyPair() (sk *SecretKey, pk *PublicKey) {
	sk = keygen.GenSecretKey()
	return sk, keygen.GenPublicKey(sk)
}

// NewRelinKey generates a new evaluation key from the provided SecretKey. It will be used to relinearize a ciphertext (encrypted under a PublicKey generated from the provided SecretKey)
// of degree > 1 to a ciphertext of degree 1. Max degree is the maximum degree of the ciphertext allowed to relinearize.
func (keygen *keyGenerator) GenRelinKey(sk *SecretKey, maxDegree uint64) (evk *EvaluationKey) {

	if keygen.ringQP == nil {
		panic("Cannot GenRelinKey: modulus P is empty")
	}

	evk = new(EvaluationKey)

	evk.evakey = make([]*SwitchingKey, maxDegree)

	keygen.polypool[0].Copy(sk.Get())

	ringQP := keygen.ringQP

	ringQP.MulCoeffsMontgomery(sk.Get(), sk.Get(), keygen.polypool[1])
	evk.evakey[0] = keygen.newSwitchingKey(keygen.polypool[1], sk.Get())

	for i := uint64(1); i < maxDegree; i++ {
		ringQP.MulCoeffsMontgomery(keygen.polypool[1], sk.Get(), keygen.polypool[1])
		evk.evakey[i] = keygen.newSwitchingKey(keygen.polypool[0], sk.Get())
	}

	keygen.polypool[0].Zero()
	keygen.polypool[1].Zero()

	return
}

// NewRelinKey creates a new EvaluationKey with zero values.
func NewRelinKey(params *Parameters, maxDegree uint64) (evakey *EvaluationKey) {

	evakey = new(EvaluationKey)

	beta := params.Beta()

	evakey.evakey = make([]*SwitchingKey, maxDegree)

	for w := uint64(0); w < maxDegree; w++ {

		evakey.evakey[w] = new(SwitchingKey)

		evakey.evakey[w].evakey = make([][2]*ring.Poly, beta)

		for i := uint64(0); i < beta; i++ {

			evakey.evakey[w].evakey[i][0] = ring.NewPoly(uint64(1<<params.logN), uint64(len(params.qi)+len(params.pi)))
			evakey.evakey[w].evakey[i][1] = ring.NewPoly(uint64(1<<params.logN), uint64(len(params.qi)+len(params.pi)))
		}
	}

	return
}

// Get returns the slice of SwitchingKeys of the target EvaluationKey.
func (evk *EvaluationKey) Get() []*SwitchingKey {
	return evk.evakey
}

// Set sets the polynomial of the target EvaluationKey as the input polynomials.
func (evk *EvaluationKey) Set(rlk [][][2]*ring.Poly) {

	evk.evakey = make([]*SwitchingKey, len(rlk))
	for i := range rlk {
		evk.evakey[i] = new(SwitchingKey)
		evk.evakey[i].evakey = make([][2]*ring.Poly, len(rlk[i]))
		for j := range rlk[i] {
			evk.evakey[i].evakey[j][0] = rlk[i][j][0].CopyNew()
			evk.evakey[i].evakey[j][1] = rlk[i][j][1].CopyNew()
		}
	}
}

// GenSwitchingKey generates a new key-switching key, that will allow to re-encrypt under the output-key a ciphertext encrypted under the input-key.
func (keygen *keyGenerator) GenSwitchingKey(skInput, skOutput *SecretKey) (newevakey *SwitchingKey) {

	if keygen.ringQP == nil {
		panic("Cannot GenRelinKey: modulus P is empty")
	}

	keygen.ringQP.Copy(skInput.Get(), keygen.polypool[0])
	newevakey = keygen.newSwitchingKey(keygen.polypool[0], skOutput.Get())
	keygen.polypool[0].Zero()
	return
}

// NewSwitchingKey returns a new SwitchingKey with zero values.
func NewSwitchingKey(params *Parameters) (evakey *SwitchingKey) {

	evakey = new(SwitchingKey)

	// delta_sk = skInput - skOutput = GaloisEnd(skOutput, rotation) - skOutput
	evakey.evakey = make([][2]*ring.Poly, params.Beta())

	for i := uint64(0); i < params.Beta(); i++ {
		evakey.evakey[i][0] = ring.NewPoly(uint64(1<<params.logN), uint64(len(params.qi)+len(params.pi)))
		evakey.evakey[i][1] = ring.NewPoly(uint64(1<<params.logN), uint64(len(params.qi)+len(params.pi)))
	}

	return
}

// NewRotationKeys returns a new empty RotationKeys struct.
func NewRotationKeys() (rotKey *RotationKeys) {
	rotKey = new(RotationKeys)
	return
}

// GenRot populates the target RotationKeys with a SwitchingKey for the desired rotation type and amount.
func (keygen *keyGenerator) GenRot(rotType Rotation, sk *SecretKey, k uint64, rotKey *RotationKeys) {

	ringQP := keygen.ringQP

	if ringQP == nil {
		panic("Cannot GenRot: modulus P is empty")
	}

	switch rotType {
	case RotationLeft:

		if rotKey.evakeyRotColLeft == nil {
			rotKey.evakeyRotColLeft = make(map[uint64]*SwitchingKey)
		}

		if rotKey.evakeyRotColLeft[k] == nil && k != 0 {
			rotKey.evakeyRotColLeft[k] = keygen.genrotKey(sk.Get(), ring.ModExp(GaloisGen, 2*ringQP.N-k, 2*ringQP.N))
		}

	case RotationRight:

		if rotKey.evakeyRotColRight == nil {
			rotKey.evakeyRotColRight = make(map[uint64]*SwitchingKey)
		}

		if rotKey.evakeyRotColRight[k] == nil && k != 0 {
			rotKey.evakeyRotColRight[k] = keygen.genrotKey(sk.Get(), ring.ModExp(GaloisGen, k, 2*ringQP.N))
		}

	case RotationRow:
		rotKey.evakeyRotRow = keygen.genrotKey(sk.Get(), 2*ringQP.N-1)
	}
}

// GenRotationKeysPow2 generates a new rotation key with all the power-of-two rotations to the left and right, as well as the conjugation.
func (keygen *keyGenerator) GenRotationKeysPow2(skOutput *SecretKey) (rotKey *RotationKeys) {

	if keygen.ringQP == nil {
		panic("Cannot GenRotationKeysPow2: modulus P is empty")
	}

	rotKey = NewRotationKeys()

	for n := uint64(1); n < 1<<(keygen.params.LogN()-1); n <<= 1 {
		keygen.GenRot(RotationLeft, skOutput, n, rotKey)
		keygen.GenRot(RotationRight, skOutput, n, rotKey)
	}

	keygen.GenRot(RotationRow, skOutput, 0, rotKey)

	return
}

// SetRotKey sets the target RotationKeys' SwitchingKey for the specified rotation type and amount with the input
// polynomials.
func (rotKey *RotationKeys) SetRotKey(rotType Rotation, k uint64, evakey [][2]*ring.Poly) {

	switch rotType {
	case RotationLeft:

		if rotKey.evakeyRotColLeft == nil {
			rotKey.evakeyRotColLeft = make(map[uint64]*SwitchingKey)
		}

		if rotKey.evakeyRotColLeft[k] == nil && k != 0 {

			rotKey.evakeyRotColLeft[k] = new(SwitchingKey)
			rotKey.evakeyRotColLeft[k].evakey = make([][2]*ring.Poly, len(evakey))
			for j := range evakey {
				rotKey.evakeyRotColLeft[k].evakey[j][0] = evakey[j][0].CopyNew()
				rotKey.evakeyRotColLeft[k].evakey[j][1] = evakey[j][1].CopyNew()
			}
		}

	case RotationRight:

		if rotKey.evakeyRotColRight == nil {
			rotKey.evakeyRotColRight = make(map[uint64]*SwitchingKey)
		}

		if rotKey.evakeyRotColRight[k] == nil && k != 0 {

			rotKey.evakeyRotColRight[k] = new(SwitchingKey)
			rotKey.evakeyRotColRight[k].evakey = make([][2]*ring.Poly, len(evakey))
			for j := range evakey {
				rotKey.evakeyRotColRight[k].evakey[j][0] = evakey[j][0].CopyNew()
				rotKey.evakeyRotColRight[k].evakey[j][1] = evakey[j][1].CopyNew()
			}
		}

	case RotationRow:

		if rotKey.evakeyRotRow == nil {

			rotKey.evakeyRotRow = new(SwitchingKey)
			rotKey.evakeyRotRow.evakey = make([][2]*ring.Poly, len(evakey))
			for j := range evakey {
				rotKey.evakeyRotRow.evakey[j][0] = evakey[j][0].CopyNew()
				rotKey.evakeyRotRow.evakey[j][1] = evakey[j][1].CopyNew()
			}
		}
	}
}

func (keygen *keyGenerator) genrotKey(sk *ring.Poly, gen uint64) (switchingkey *SwitchingKey) {

	skIn := sk
	skOut := keygen.polypool[1]

	ring.PermuteNTT(skIn, gen, skOut)

	switchingkey = keygen.newSwitchingKey(skIn, skOut)

	keygen.polypool[0].Zero()
	keygen.polypool[1].Zero()

	return
}

func (keygen *keyGenerator) newSwitchingKey(skIn, skOut *ring.Poly) (switchingkey *SwitchingKey) {

	switchingkey = new(SwitchingKey)

	ringQP := keygen.ringQP

	alpha := keygen.params.Alpha()
	beta := keygen.params.Beta()

	var index uint64

	// delta_sk = skIn - skOut = GaloisEnd(skOut, rotation) - skOut

	ringQP.MulScalarBigint(skIn, keygen.pBigInt, keygen.polypool[0])

	switchingkey.evakey = make([][2]*ring.Poly, beta)

	for i := uint64(0); i < beta; i++ {

		// e
		switchingkey.evakey[i][0] = keygen.gaussianSampler.ReadNew()
		ringQP.NTTLazy(switchingkey.evakey[i][0], switchingkey.evakey[i][0])
		ringQP.MForm(switchingkey.evakey[i][0], switchingkey.evakey[i][0])
		// a
		switchingkey.evakey[i][1] = keygen.uniformSampler.ReadNew()

		// e + skIn * (qiBarre*qiStar) * 2^w
		// (qiBarre*qiStar)%qi = 1, else 0

		for j := uint64(0); j < alpha; j++ {

			index = i*alpha + j

			qi := ringQP.Modulus[index]
			p0tmp := keygen.polypool[0].Coeffs[index]
			p1tmp := switchingkey.evakey[i][0].Coeffs[index]

			for w := uint64(0); w < ringQP.N; w++ {
				p1tmp[w] = ring.CRed(p1tmp[w]+p0tmp[w], qi)
			}

			// Handles the case where nb pj does not divide nb qi
			if index >= keygen.params.QiCount() {
				break
			}

		}

		// skIn * (qiBarre*qiStar) * 2^w - a*sk + e
		ringQP.MulCoeffsMontgomeryAndSub(switchingkey.evakey[i][1], skOut, switchingkey.evakey[i][0])
	}

	return
}
