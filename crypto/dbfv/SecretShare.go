package dbfv

import (
	"encoding/binary"

	//"github.com/ldsec/lattigo/v2/bfv"
	//"github.com/ldsec/lattigo/v2/ring"
	//"github.com/ldsec/lattigo/v2/utils"
	"Vote/crypto/bfv"
	"Vote/crypto/ring"
	"Vote/crypto/utils"
	"math/big"
	"math/bits"
	"math/rand"
)

// 参与方根据自己的部分解密结果和其他参与方的部分解密结果恢复出最终的明文

type DKGProtocol struct {
	params *bfv.Parameters
	N      uint64
	t      uint64
	ID     uint64 // 该投票人的ID,1,2,3
	//matrix [][]uint64  //储存自己和他人发来的信息（多项式带入i的结果---一个向量）拼接成的矩阵
	//第i个人的信息放在第i列（生成时可先放在第i行，再进行转置）

	ringQP          *ring.Ring
	pBigInt         *big.Int
	polypool        [2]*ring.Poly
	gaussianSampler *ring.GaussianSampler
	uniformSampler  *ring.UniformSampler
	result          [][]uint64
}

type DKGShare struct {
	ID    uint64
	share []uint64
}

// PublicKey is a structure that stores the PublicKey.
type PublicKey struct {
	pk [2]*ring.Poly
}

func NewDKGProtocol(params *bfv.Parameters, N, t uint64, ID uint64) *DKGProtocol {
	var ringQP *ring.Ring
	var err error
	if ringQP, err = ring.NewRing(params.N(), append(params.Qi(), params.Pi()...)); err != nil {
		panic(err)
	}

	var pBigInt *big.Int
	if len(params.Pi()) != 0 {
		pBigInt = ring.NewUint(1)
		for _, pi := range params.Pi() {
			pBigInt.Mul(pBigInt, ring.NewUint(pi))
		}
	}

	prng, err := utils.NewPRNG()
	if err != nil {
		panic(err)
	}
	dkg := new(DKGProtocol)
	dkg.params = params.Copy()
	dkg.ringQP = ringQP
	dkg.pBigInt = pBigInt
	dkg.polypool = [2]*ring.Poly{ringQP.NewPoly(), ringQP.NewPoly()}
	dkg.gaussianSampler = ring.NewGaussianSampler(prng, ringQP, params.Sigma(), uint64(6*params.Sigma()))
	dkg.uniformSampler = ring.NewUniformSampler(prng, ringQP)
	dkg.N = N
	dkg.t = t
	dkg.ID = ID

	return dkg
}

/*
   计算多项式
*/
func (dkg *DKGProtocol) poly(q, x uint64, index []int64) uint64 {

	l := len(index)

	big1 := new(big.Int).SetUint64(q)
	//result := new(big.Int).SetUint64(uint64(0))
	//
	//for i := 0; i < l; i++ {
	//	ip := big.NewInt(1)
	//	temp1 := new(big.Int).SetInt64(int64(index[i]))
	//	temp2 := new(big.Int).SetUint64(x)
	//	temp3 := new(big.Int).SetInt64(int64(i))
	//	temp2.Exp(temp2, temp3, big1)
	//	ip.Mul(temp1, temp2)
	//	result.Add(result, ip)
	//}
	temp := new(big.Int).SetUint64(uint64(index[l-1]))
	temp1 := new(big.Int).SetUint64(x)

	for i := l - 1; i >= 1; i-- {

		temp2 := new(big.Int).SetUint64(uint64(index[i-1]))
		temp.Mul(temp, temp1)
		temp.Add(temp, temp2)
		//result.Add(result,temp)
	}

	return temp.Mod(temp, big1).Uint64()
	//return result.Mod(result, big1).Uint64()
}

func (dkg *DKGProtocol) AllocateDKGShare(share []uint64) DKGShare {
	return DKGShare{dkg.ID, share}
}

/*
生成对应的share
*/
func (dkg *DKGProtocol) genShare(secret uint64, q uint64) []uint64 {

	//储存最终结果,n份share
	var result []uint64
	//构造多项式系数
	var index []int64

	var secret2 = int64(secret)
	var q2 = int64(q)
	index = append(index, secret2)
	for i := uint64(1); i < dkg.t; i++ {
		index = append(index, rand.Int63n(q2))
	}
	//计算share
	for i := uint64(1); i <= dkg.N; i++ {
		result = append(result, dkg.poly(q, i, index)%q)
	}
	return result
}

/*
生成拉格朗日系数
*/
func (dkg *DKGProtocol) genCoffee(index []int64, q uint64) []uint64 {

	var coffee []uint64
	var l = len(index)

	mod := new(big.Int).SetUint64(q)

	for i := 0; i < l; i++ {
		nume := new(big.Int).SetInt64(int64(1))
		deno := new(big.Int).SetInt64(int64(1))

		for j := 0; j < l; j++ {
			if j != i {
				nume.Mul(nume, new(big.Int).SetInt64(-index[j]))
				deno.Mul(deno, new(big.Int).SetInt64(index[i]-index[j]))
			}
		}

		nume.Mod(nume, mod)
		deno.ModInverse(deno, mod)

		var temp = nume.Mul(nume, deno)

		temp.Mod(temp, mod)
		coffee = append(coffee, temp.Uint64())

	}
	return coffee
}

// 拉格朗日系数，一行代表一个参与方
func (dkg *DKGProtocol) GenCoffee(index []int64) map[uint64][]uint64 {

	coeffs := make([][]uint64, len(index))

	for i := 0; i < len(index); i++ {
		coeffs[i] = make([]uint64, dkg.params.QPiCount())
	}
	for i := uint64(0); i < dkg.params.QiCount(); i++ {
		tmp_coeffs := dkg.genCoffee(index, dkg.params.Moduli().Qi[i])
		for j := 0; j < len(index); j++ {
			coeffs[j][i] = tmp_coeffs[j]
		}
	}

	for i := uint64(0); i < dkg.params.PiCount(); i++ {
		tmpCoeffs := dkg.genCoffee(index, dkg.params.Moduli().Pi[i])
		for j := 0; j < len(index); j++ {
			coeffs[j][i+dkg.params.QiCount()] = tmpCoeffs[j]
		}
	}
	ans := make(map[uint64][]uint64)
	for i := 0; i < len(index); i++ {
		ans[uint64(index[i])] = coeffs[i]
	}
	return ans
}

/*
单点生成要传输的信息--一个矩阵，该矩阵的第i行表示此人要发给第i个人的信息
注意第i个人的第i行不转给他人，留给自己
*/

func (dkg *DKGProtocol) innerLoop(count int, flag int, i uint64, c chan tmpMatrix) {
	q := uint64(0)
	m := tmpMatrix{
		index: i,
	}
	for j := uint64(0); j < dkg.params.QPiCount(); j++ {
		if j < dkg.params.QiCount() {
			q = dkg.params.Qi()[j]
		} else {
			q = dkg.params.Pi()[j-dkg.params.QiCount()]
		}

		var secret []uint64

		if flag == 0 {
			secret = dkg.zeros(count, q)
		} else if flag == 1 {
			secret = dkg.ones(count, q)
		} else {
			secret = dkg.minus(count, q)
		}
		share := dkg.genShare(secret[dkg.ID-1], q)
		m.matrix = append(m.matrix, share)
	}
	c <- m
}

type tmpMatrix struct {
	index  uint64
	matrix [][]uint64
}

func (dkg *DKGProtocol) DKGGenMatrixSingle() (ans []DKGShare) {

	dkg.result = make([][]uint64, dkg.params.N()*dkg.params.QPiCount())
	for i := uint64(0); i < dkg.params.N()*dkg.params.QPiCount(); i++ {
		dkg.result[i] = make([]uint64, dkg.N)
	}

	randomList := make([]int, dkg.params.N())
	rand.Seed(45)
	for i := uint64(0); i < dkg.params.N(); i++ {
		randomList[i] = int(i % dkg.N)
	}

	c := make(chan tmpMatrix, 2000)
	for i := uint64(0); i < dkg.params.N(); i++ {
		flag := rand.Intn(3)
		go dkg.innerLoop(randomList[i], flag, i, c)
	}
	count := uint64(0)
	for {
		select {
		case m := <-c:
			count += dkg.params.QPiCount()
			index := m.index
			matrix := m.matrix
			for k := uint64(0); k < dkg.params.QPiCount(); k++ {
				dkg.result[index*dkg.params.QPiCount()+k] = matrix[k]
			}
		default:
		}
		if count == dkg.params.N()*dkg.params.QPiCount() {
			break
		}
	}

	//矩阵转置
	tmpans := make([][]uint64, len(dkg.result[0]))
	col := len(dkg.result)

	for i := 0; i < len(dkg.result[0]); i++ {
		tmpans[i] = make([]uint64, col)
	}

	for i := 0; i < len(dkg.result); i++ {
		for j := 0; j < len(dkg.result[0]); j++ {
			tmpans[j][i] = dkg.result[i][j]
		}
	}

	ans = make([]DKGShare, dkg.N)
	for i := uint64(0); i < dkg.N; i++ {
		ans[i] = dkg.AllocateDKGShare(tmpans[i])
	}
	return ans
}

/*
根据获取到他人的信息来还原出sk_i的蒙哥马利表示形式
接受到一个矩阵，第i列就是第i个人的多项式带入i的结果
*/
// 参数表示一个share数组，数组中的每一个元素都是
func (dkg *DKGProtocol) Combine(dkgShare []DKGShare) *bfv.SecretKey {

	result := make([][]uint64, dkg.params.N())

	for i := uint64(0); i < dkg.params.QPiCount(); i++ {
		result[i] = make([]uint64, dkg.params.QPiCount())
	}
	var b = 0
	var q uint64 = 0
	var temp []uint64

	for i := uint64(0); i < dkg.params.QPiCount()*dkg.params.N(); i++ {

		var j = i % dkg.params.QPiCount()
		if j < dkg.params.QiCount() {
			q = dkg.params.Qi()[j]
		} else {
			q = dkg.params.Pi()[j-dkg.params.QiCount()]
		}
		ans := uint64(0)
		for k := 0; k < len(dkgShare); k++ {
			ans += dkgShare[k].share[i] % q
		}
		temp = append(temp, ans%q)
		if j == dkg.params.QPiCount()-1 {
			result[b] = temp
			b++
			temp = []uint64{}
		}
	}

	//矩阵转置
	ans := make([][]uint64, len(result[0]))
	col := len(result)
	for i := 0; i < len(result[0]); i++ {
		ans[i] = make([]uint64, col)
	}

	for i := 0; i < len(result); i++ {
		for j := 0; j < len(result[0]); j++ {
			ans[j][i] = result[i][j]
		}
	}
	sk := dkg.newSecretKeyFromMForm(ans)
	return sk
}

/*
辅助函数，用来计算数组的和
*/
func (dkg *DKGProtocol) sum(its []uint64) uint64 {

	var result uint64 = 0
	var l = len(its)
	for i := 0; i < l; i++ {
		result += its[i]
	}
	return result
}

/*
1对应的蒙哥马利表示
*/
func (dkg *DKGProtocol) ones(count int, q uint64) []uint64 {

	var result = make([]uint64, dkg.N)
	var param = dkg.bRedParams1(q)
	result[count] = dkg.mForm1(1, q, param)
	return result
}

func (dkg *DKGProtocol) zeros(count int, q uint64) []uint64 {

	var result = make([]uint64, dkg.N)
	var param = dkg.bRedParams1(q)
	result[count] = dkg.mForm1(0, q, param)

	return result
}

func (dkg *DKGProtocol) minus(count int, q uint64) []uint64 {

	var result = make([]uint64, dkg.N)
	var param = dkg.bRedParams1(q)
	result[count] = dkg.mForm1(q-1, q, param)
	return result
}

func (dkg *DKGProtocol) mForm1(a, q uint64, u []uint64) (r uint64) {
	mhi, _ := bits.Mul64(a, u[1])
	r = -(a*u[0] + mhi) * q
	if r >= q {
		r -= q
	}
	return
}

func (dkg *DKGProtocol) bRedParams1(q uint64) (params []uint64) {
	bigR := new(big.Int).Lsh(dkg.newUint1(1), 128)
	bigR.Quo(bigR, dkg.newUint1(q))

	mhi := new(big.Int).Rsh(bigR, 64).Uint64()
	mlo := bigR.Uint64()

	return []uint64{mhi, mlo}
}

func (dkg *DKGProtocol) newUint1(v uint64) *big.Int {
	return new(big.Int).SetUint64(v)
}

// 利用一个蒙哥马利表示形式的数组生成一个私钥sk
func (dkg *DKGProtocol) newSecretKeyFromMForm(coffees [][]uint64) (sk *bfv.SecretKey) {
	sk = new(bfv.SecretKey)
	tmpPoly := dkg.ringQP.NewPoly()
	tmpPoly.SetCoefficients(coffees)
	dkg.ringQP.NTT(tmpPoly, tmpPoly)
	sk.Set(tmpPoly)
	return
}

func (dkg *DKGProtocol) MarshalBinary(share DKGShare) (data []byte) {
	dataLen := (len(share.share) + 1) << 3 // 没有包含多项式的字节数组大小

	data = make([]byte, dataLen)
	binary.BigEndian.PutUint64(data[0:8], share.ID)
	for i := 0; i < len(share.share); i++ {
		binary.BigEndian.PutUint64(data[8+i<<3:8+(i+1)<<3], share.share[i])
	}
	return data
}

// UnmarshalBinary decodes a slice of byte on the target polynomial.
func (dkg *DKGProtocol) UnmarshalBinary(data []byte) (share DKGShare) {

	share.ID = binary.BigEndian.Uint64(data[0:8])
	length := len(data)>>3 - 1
	share.share = make([]uint64, length)
	for i := 0; i < length; i++ {
		share.share[i] = binary.BigEndian.Uint64(data[8+i<<3 : 8+(i+1)<<3])
	}
	return share
}
