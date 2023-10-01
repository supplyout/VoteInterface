package message

import (
	. "Vote/crypto/party"
	. "Vote/vote"
)

const (
	RootTopic1 = "P2P_this_is_my_root_topic"
)

func GenVoteBrief(v Vote) Message {
	msg := &VoteBrief{
		Type:      VoteBriefType,
		From:      v.From,
		To:        RootTopic1,
		Topic:     v.Topic,
		Title:     v.Title,
		Brief:     v.Brief,
		VoteType:  v.VoteType,
		StartTime: v.StartTime,
		Deadline:  v.Deadline,
	}
	return msg
}

func GenVoteDetail(v Vote) Message {
	msg := &VoteDetail{
		Type:       VoteDetailType,
		From:       v.From,
		To:         v.Topic,
		Topic:      v.Topic,
		Title:      v.Title,
		Brief:      v.Brief,
		VoteType:   v.VoteType,
		Candidates: v.Candidates,
	}
	return msg
}

func GenVoteParams(v Vote, p Poll) Message {
	a, _ := p.A.MarshalBinary()
	params, _ := p.Params.MarshalBinary()
	msg := &VoteParams{
		Type:   VoteParamsType,
		From:   v.From,
		To:     v.Topic,
		Topic:  v.Topic,
		N:      uint64(len(v.Parties)),
		T:      p.T,
		A:      a,
		Params: params,
	}
	return msg
}

func GenVoteResult(v Vote, from string) Message {
	msg := &VoteResult{
		Type:   VoteResultType,
		From:   from,
		To:     v.Topic,
		Topic:  v.Topic,
		Result: v.Result,
	}
	return msg
}

func GenVoteAcceptation(v Vote, acp bool, from string) Message {
	msg := &VoteAcceptation{
		Type:   VoteAcceptationType,
		From:   from,
		To:     v.From,
		Topic:  v.Topic,
		Result: acp,
	}
	return msg
}

func GenPolyResult(v Vote, from string, to string, result []byte) Message {
	msg := &PolyResult{
		Type:   PolyResultType,
		From:   from,
		To:     to,
		Topic:  v.Topic,
		Result: result,
	}
	return msg
}

func GenPubKeyPart(v Vote, from string, part []byte) Message {
	msg := &PubKeyPart{
		Type:  PubKeyPartType,
		From:  from,
		To:    v.Topic,
		Topic: v.Topic,
		Part:  part,
	}
	return msg
}

//func GenPubKey(v Vote, from string, key []byte) Message {
//	msg := &PubKey{
//		Type:  PubKeyType,
//		From:  from,
//		To:    v.Topic,
//		Topic: v.Topic,
//		Key:   key,
//	}
//	return msg
//}

func GenPlainTextPart(v Vote, from string, text []byte) Message {
	msg := &PlainTextPart{
		Type:  PlainTextPartType,
		From:  from,
		To:    v.Topic,
		Topic: v.Topic,
		Text:  text,
	}
	return msg
}

func GenCipherText(v Vote, from string, text []byte) Message {
	msg := &CipherText{
		Type:  CipherTextType,
		From:  from,
		To:    v.Topic,
		Topic: v.Topic,
		Text:  text,
	}
	return msg
}

func GenPersonID(v Vote, from string, text []byte) Message {
	msg := &ID{
		Type:  IDType,
		From:  from,
		To:    v.Topic,
		Topic: v.Topic,
		Text:  text,
	}
	return msg
}
