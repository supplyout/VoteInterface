package ui

/*
存疑，生成pk后，接受类似候选人名单等的投票信息
*/
/*
func (s *server) handleVoteDetail(ctx context.Context, msg Message) {
	m := msg.(*VoteDetail)
	// 忽略自己的广播信息
	if m.From == s.Node.GetNodeID() {
		return
	}
	if s.Stat != NodeStatGotPK {
		Logger.Infof("当前发起节点未处接收他人确定结果状态，无法进行统计")
		return
	}

	s.currentVote.Title = m.Title
	s.currentVote.Candidates = m.Candidates

	//改变状态
	s.Stat = NodeStatGotDetail
	Logger.Infof("当前状态:NodeStatGotDetail")

	//跳转到投票界面

	msg4K := MsgVoteDetails{
		MType:      MsgVoteDetailsType,
		Title:      m.Title,
		Brief:      m.Brief,
		Candidates: m.Candidates,
	}
	s.SendMsgToKot(&msg4K)
}
*/
/*
func (s *server) handlePubKeyPart(ctx context.Context, msg Message) {
	m := msg.(*PubKeyPart)
	if s.Stat != NodeStatSentPki {
		Logger.Infof("当前发起节点未处接收他人确定结果状态，无法进行统计")
		return
	}
	//解码得到他人发过来的pk_i
	Logger.Infof("收到pki")
	pki := s.currentParty.UnMarshalPki(m.Part)
	s.currentVote.PartPubKey = append(s.currentVote.PartPubKey, pki)

	//获得t个pk_i之后

	if len(s.currentVote.PartPubKey) >= len(s.currentVote.Parties) {
		//生成完整的pk
		s.currentParty.GenPk(s.currentVote.PartPubKey)
		//更新等待界面的消息


		s.Stat = NodeStatGotPK
		Logger.Infof("当前状态:NodeStatGotPK")
	}

}
*/
/*
获取他人发来的信息，用于生成ski，并用ski生成pki
*/
/*
func (s *server) handlePolyResult(ctx context.Context, msg Message) {
	m := msg.(*PolyResult)
	if s.Stat != NodeStatGotParams {
		Logger.Infof("当前发起节点未处接收他人确定结果状态，无法进行统计")
		return
	}

	share := s.currentParty.UnmarshalSecretShare(m.Result)

	s.currentVote.SkInf = append(s.currentVote.SkInf, share)

	//判断获得其他所有人的信息
	log.Printf("len:%d", len(s.currentVote.Parties))
	if len(s.currentVote.Parties) == len(s.currentVote.SkInf) {

		//生成每个人的SKi
		s.currentParty.GenSki(s.currentVote.SkInf)
		//dialog.ShowInformation("成功生成sk_i", "成功生成sk_i", s.currentPage.GetWindow())
		Logger.Infof("成功生成sk_i")
		//生成pki
		pki := s.currentParty.GenPki()
		//s.currentVote.PartPubKey = append(s.currentVote.PartPubKey, pki) //将自己的pki保存到Vote中
		Logger.Infof("成功生成pk_i")
		//将pki广播
		pkiMarshaled := s.currentParty.MarshalPki(pki)
		s.MsgCh[Write] <- GenPubKeyPart(*s.currentVote, s.Node.GetNodeID(), pkiMarshaled)
		s.Stat = NodeStatSentPki
		Logger.Infof("当前状态:NodeStatSentPki")
		//更新等待界面的消息

	}

}
*/

/*
接受他人发来的部分解密的信息
*/
/*
func (s *server) handlePlainText(ctx context.Context, msg Message) {
	m := msg.(*PlainTextPart)
	if s.Stat != NodeStatSentCts && s.Stat!= NodeStatSentID {
		Logger.Infof("进行最终的结果，无法进行统计")
		return
	}
	Logger.Infof("收到别人部分解密的结果")
	//解码部分解密的信息
	partdec := s.currentParty.UnMarshalCDShare(m.Text)

	//将部分解密结果存入此应用
	s.currentVote.Partdec = append(s.currentVote.Partdec, partdec)
	s.currentVote.DecParties = append(s.currentVote.DecParties, m.From)

	//判断a.Partdec的长度是否>=t，若满足条件，则可以进行Findec
	if len(s.currentVote.Partdec) == s.currentVote.LengthDec {
		Logger.Infof("进行最终解密")
		s.currentVote.Result = make(map[string]int)
		//保存投票最终结果的票数
		fin := s.currentParty.FinaDec(s.currentVote.Partdec)
		Logger.Infof("result:%v", fin)

		//按对应的顺序将候选人和票数一一对应存入result
		l := len(s.currentVote.Candidates)
		for i := 0; i < l; i++ {
			s.currentVote.Result[s.currentVote.Candidates[i]] = int(fin[i])
		}
		Logger.Infof("最终解密成功")
		s.MsgCh[Write] <- GenVoteResult(*s.currentVote, s.Node.GetNodeID())
		s.Stat = NodeStatSentResult
		Logger.Infof("当前状态:NodeStatSentResult")

	}
	//s.MsgCh[Write] <- GenVoteResult(*s.currentVote, s.Node.GetNodeID())
	//s.Stat = NodeStatGotPts
}
*/

/*
接受他人的密文，对这些密文进行运算
*/
/*
func (s *server) handleCipherText(ctx context.Context, msg Message) {
	m := msg.(*CipherText)
	Logger.Infof("收到他人的密文")
	if s.Stat != NodeStatVoted && s.Stat != NodeStatGotDetail {
		Logger.Infof("当前发起节点未处于接收他人确定结果状态，无法进行统计")
		return
	}
	//解码得到他人发过来的密文
	cipher := s.currentParty.UnMarshalCiphertext(m.Text)

	s.currentVote.EvalCipher = append(s.currentVote.EvalCipher, cipher)

	//判断获得其他所有人的信息
	if len(s.currentVote.Parties) == len(s.currentVote.EvalCipher) {

		//对密文运算
		s.currentParty.EvalAdd(s.currentVote.EvalCipher)

		////部分解密
		//partdec := s.currentParty.PartDec()
		//
		////发广播
		//s.MsgCh[Write] <- GenPlainTextPart(*s.currentVote, s.Node.GetNodeID(), s.currentParty.MarshalCDShare(partdec))
		////s.MsgCh[Write] <- Gen
		//if
		rand.Seed(int64(s.currentParty.ID))
		randnum := rand.Intn(10)

		if randnum <= 7 {
			text := s.currentParty.MarshalID(s.currentParty.ID)
			Logger.Infof("当前节点参与投票 %s", text)
			s.MsgCh[Write] <- GenPersonID(*s.currentVote, s.Node.GetNodeID(), text)
			s.currentVote.Flag = true
		} else {
			s.MsgCh[Write] <- GenPersonID(*s.currentVote, s.Node.GetNodeID(), s.currentParty.MarshalID(0))
			Logger.Info("此节点不参与解密")
			s.currentVote.Flag = false
		}

		s.Stat = NodeStatSentID
		Logger.Infof("当前状态:NodeStatSentID")
	}

}
*/

/*
接受他人的ID，计算出对应的拉格朗日系数，进行部分解密
*/
/*
func (s *server) handleCoffin(ctx context.Context, msg Message) {
	m := msg.(*ID)
	Logger.Infof("收到ID")
	if s.Stat != NodeStatSentID && s.Stat!=NodeStatVoted{
		Logger.Infof("当前发起节点进行系数的计算，无法进行统计")
		return
	}

	//解码得到他人发过来的ID
	id := s.currentParty.UnmarshalID(m.Text)
	s.currentVote.DecPartNum = append(s.currentVote.DecPartNum, id)

	//cipher := s.currentParty.UnMarshalCiphertext(m.Text)
	//如果当前节点参与投票，进行部分解密
	if s.currentVote.Flag {

		//判断获得其他所有人的信息
		if len(s.currentVote.Parties) == len(s.currentVote.DecPartNum) {

			//将trueid作为参数传入部分解密的结果
			var trueid []int64
			for i := 0; i < len(s.currentVote.DecPartNum); i++ {
				if s.currentVote.DecPartNum[i] > 0 {
					trueid = append(trueid, int64(s.currentVote.DecPartNum[i]))
				}
			}
			s.currentVote.LengthDec = len(trueid)
			////部分解密
			//println("输出trueid长度: ", len(trueid))
			//fmt.Println(trueid)
			partdec := s.currentParty.PartDec(trueid)
			//
			//发广播
			s.MsgCh[Write] <- GenPlainTextPart(*s.currentVote, s.Node.GetNodeID(), s.currentParty.MarshalCDShare(partdec))
			s.Stat = NodeStatSentCts
			Logger.Infof("当前状态:NodeStatSentCts")
		}
	} else {
		s.Stat = NodeStatSentCts
		Logger.Infof("当前状态:NodeStatSentCts")
	}

}
*/
