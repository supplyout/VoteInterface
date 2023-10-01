package party
//
//import (
//	"Vote/crypto/dbfv"
//)
//
//func main() {
//
//	poll := NewPoll(4, 3) // 表示一次投票
//
//	pollData := poll.MarshalBinary()
//	pollNew := new(Poll)
//	pollNew.UnmarshalBinary(pollData)
//
//	participant := make([]*Party, 4)
//	for i := 0; i < 4; i++ {
//		participant[i] = NewParty(uint64(i+1), pollNew)
//	}
//
//	//
//	Share1 := make([][]dbfv.DKGShare, poll.N)
//	for i := uint64(0); i < poll.N; i++ {
//		Share1[i] = make([]dbfv.DKGShare, poll.N)
//	}
//
//	for i := 0; i < 4; i++ {
//		share := participant[i].GenSecretShare() // []dbfv.DKGShare
//
//		for j := 0; j < 4; j++ {
//			MShare := participant[i].MarshalSecretShare(share[j])
//			Share1[j][i] = participant[j].UnmarshalSecretShare(MShare)
//
//		}
//	}
//
//	for i := 0; i < 4; i++ {
//
//		participant[i].GenSki(Share1[i])
//	}
//
//	/*
//		index := make([]int, 2)
//		index[0] = 1
//		index[1] = 2
//		//index[2] = 3
//		//index[3] = 4
//		coeffs := participant[0].genSecretKey.GenCoffee(index)  // ok
//
//		keygen := bfv.NewKeyGenerator(Params)
//		lambdaPoly1 := keygen.NewPolyFromCoeffs(coeffs[0])
//		lambdaPoly2 := keygen.NewPolyFromCoeffs(coeffs[1])
//		//lambdaPoly3 := keygen.NewPolyFromCoeffs(coeffs[2])
//		//lambdaPoly4 := keygen.NewPolyFromCoeffs(coeffs[3])
//
//		tmp1 := keygen.Mul(lambdaPoly1, participant[0].ski.Get())
//		tmp2 := keygen.Mul(lambdaPoly2, participant[1].ski.Get())
//		//tmp3 := keygen.Mul(lambdaPoly3, participant[2].ski.Get())
//		//tmp4 := keygen.Mul(lambdaPoly4, participant[3].ski.Get())
//
//		tmp := keygen.Add(tmp1, tmp2)     // tmp = lambda1 * sk1 + lambda2 * sk2
//		//tmpp := keygen.Add(tmp3, tmp4)
//		//skPoly := keygen.Add(tmp, tmpp)
//
//		sk := keygen.GenSecretKey()
//		sk.Set(tmp)        // 将私钥设置成tmp多项式
//
//		keygen.Inv(sk.Get())   // 逆NTT变换->逆蒙哥马利变换
//		println("多项式模数")
//		println(Params.Qi()[0], " ", Params.Qi()[1], " ", Params.Pi()[0])
//
//		for i := uint64(0); i < Params.N(); i++{
//			if (sk.Get().Coeffs[0][i] != 0 && sk.Get().Coeffs[0][i] != 1 && sk.Get().Coeffs[0][i] != 34359754752){
//				println(sk.Get().Coeffs[0][i])
//			}
//		}
//	*/
//
//	Share2 := make([][]dbfv.CKGShare, poll.N)
//	for i := uint64(0); i < poll.N; i++ {
//		Share2[i] = make([]dbfv.CKGShare, poll.N)
//	}
//
//	for i := 0; i < 4; i++ {
//		share := participant[i].GenPki()
//		Mshare := participant[i].MarshalPki(share)
//		for j := 0; j < 4; j++ {
//			Share2[j][i] = participant[j].UnMarshalPki(Mshare)
//		}
//	}
//
//	for i := 0; i < 4; i++ {
//		participant[i].GenPk(Share2[i])
//	}
//
//	// 1,2,3 投 0号
//	// 4 投 1 号
//	Share3 := make([][]dbfv.CiphertextShare, poll.N)
//	for i := uint64(0); i < poll.N; i++ {
//		Share3[i] = make([]dbfv.CiphertextShare, poll.N)
//	}
//	ticket := make([]uint64, 5)
//	ticket[0] = 1
//	for i := 0; i < 3; i++ {
//		c := participant[i].Encrypt(ticket)
//		Mciphertext := participant[i].MarshalCiphertext(c)
//		for j := 0; j < 4; j++ {
//			Share3[j][i] = participant[j].UnMarshalCiphertext(Mciphertext)
//		}
//	}
//	ticket[0] = 0
//	ticket[1] = 1
//	c := participant[3].Encrypt(ticket)
//	Mciphertext := participant[3].MarshalCiphertext(c)
//	for j := 0; j < 4; j++ {
//		Share3[j][3] = participant[j].UnMarshalCiphertext(Mciphertext)
//	}
//
//	// 计票阶段
//	for i := 0; i < 4; i++ {
//		participant[i].EvalAdd(Share3[i])
//	}
//
//	// 部分解密阶段
//	Share4 := make([][]dbfv.CDShare, poll.N)
//	for i := uint64(0); i < poll.N; i++ {
//		Share4[i] = make([]dbfv.CDShare, poll.N)
//	}
//	for i := 0; i < 4; i++ {
//		share := participant[i].PartDec()
//		Mshare := participant[i].MarshalCDShare(share)
//		for j := 0; j < 4; j++ {
//			Share4[j][i] = participant[j].UnMarshalCDShare(Mshare)
//		}
//	}
//	println("计票结果")
//	for i := 0; i < 4; i++ {
//		ans := participant[i].FinaDec(Share4[i])
//		println(ans[0], " ", ans[1], " ", ans[2], " ", ans[3], " ", ans[4])
//	}
//}
