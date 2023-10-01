package message

import (
	. "Vote/log"
	"strconv"
	"time"
)

type MsgType int

const (
	VoteBriefType MsgType = iota + 100 // broadcast
	VoteDetailType
	VoteAcceptationType
	VoteResultType
	VoteParamsType
	PolyResultType
	PubKeyPartType
	PubKeyType
	PlainTextPartType
	CipherTextType
	TestMessageType
	IDType
)

// Message ui需要通过node向网络中发送的信息
type Message interface {
	GetType() MsgType
	GetFrom() string
	GetTo() string
	GetTopic() string
	MarshalJSON() ([]byte, error)
	UnmarshalJSON([]byte) error
}

type VoteBrief struct {
	Type      MsgType
	From      string
	To        string
	Title     string
	Brief     string
	VoteType  string
	Topic     string
	StartTime time.Time
	Deadline  time.Time
}

func (v VoteBrief) GetTopic() string {
	return v.Topic
}

func (v VoteBrief) GetFrom() string {
	return v.From
}

func (v VoteBrief) GetTo() string {
	return v.To
}

func (v VoteBrief) GetType() MsgType {
	return v.Type
}

type VoteParams struct {
	Type   MsgType
	From   string
	To     string
	Topic  string
	N      uint64
	T      uint64
	A      []byte
	Params []byte
}

func (v VoteParams) GetTopic() string {
	return v.Topic
}
func (v VoteParams) GetFrom() string {
	return v.From
}

func (v VoteParams) GetTo() string {
	return v.To
}

func (v VoteParams) GetType() MsgType {
	return v.Type
}

type VoteAcceptation struct {
	Type   MsgType
	From   string
	To     string
	Result bool
	Topic  string
}

func (v VoteAcceptation) GetTopic() string {
	return v.Topic
}
func (v VoteAcceptation) GetFrom() string {
	return v.From
}

func (v VoteAcceptation) GetTo() string {
	return v.To
}

func (v VoteAcceptation) GetType() MsgType {
	return v.Type
}

type VoteResult struct {
	Type   MsgType
	From   string
	To     string
	Topic  string
	Result map[string]int
}

func (v VoteResult) GetTopic() string {
	return v.Topic
}
func (v VoteResult) GetFrom() string {
	return v.From
}

func (v VoteResult) GetTo() string {
	return v.To
}

func (v VoteResult) GetType() MsgType {
	return v.Type
}

type VoteDetail struct {
	Type       MsgType
	From       string
	To         string
	Brief      string
	Topic      string
	Title      string
	VoteType   string
	Candidates []string
}

func (v VoteDetail) GetTopic() string {
	return v.Topic
}
func (v VoteDetail) GetFrom() string {
	return v.From
}

func (v VoteDetail) GetTo() string {
	return v.To
}

func (v VoteDetail) GetType() MsgType {
	return v.Type
}

type PolyResult struct {
	Type   MsgType
	From   string
	To     string
	Topic  string
	Result []byte
}

func (p PolyResult) GetTopic() string {
	return p.Topic
}
func (p PolyResult) GetFrom() string {
	return p.From
}

func (p PolyResult) GetTo() string {
	return p.To
}

func (p PolyResult) GetType() MsgType {
	return p.Type
}

type PubKeyPart struct {
	Type  MsgType
	From  string
	To    string
	Topic string
	Part  []byte
}

func (p PubKeyPart) GetTopic() string {
	return p.Topic
}
func (p PubKeyPart) GetFrom() string {
	return p.From
}
func (p PubKeyPart) GetTo() string {
	return p.To
}
func (p PubKeyPart) GetType() MsgType {
	return p.Type
}

type PubKey struct {
	Type  MsgType
	From  string
	To    string
	Topic string
	Key   []byte
}

func (p PubKey) GetTopic() string {
	return p.Topic
}
func (p PubKey) GetFrom() string {
	return p.From
}

func (p PubKey) GetTo() string {
	return p.To
}

func (p PubKey) GetType() MsgType {
	return p.Type
}

type PlainTextPart struct {
	Type  MsgType
	From  string
	To    string
	Topic string
	Text  []byte
}

func (p PlainTextPart) GetTopic() string {
	return p.Topic
}
func (p PlainTextPart) GetFrom() string {
	return p.From
}

func (p PlainTextPart) GetTo() string {
	return p.To
}

func (p PlainTextPart) GetType() MsgType {
	return p.Type
}

type CipherText struct {
	Type  MsgType
	From  string
	To    string
	Topic string
	Text  []byte
}

func (c CipherText) GetTopic() string {
	return c.Topic
}
func (c CipherText) GetFrom() string {
	return c.From
}

func (c CipherText) GetTo() string {
	return c.To
}

func (c CipherText) GetType() MsgType {
	return c.Type
}

type TestMessage struct {
	Type  MsgType
	From  string
	To    string
	Topic string
}

func (t TestMessage) GetTopic() string {
	return t.Topic
}

func (t TestMessage) GetFrom() string {
	return t.From
}

func (t TestMessage) GetTo() string {
	return t.To
}

func (t TestMessage) GetType() MsgType {
	return t.Type
}

type ID struct {
	Type      MsgType
	From      string
	To        string
	Title     string
	Brief     string
	Topic     string
	StartTime time.Time
	Deadline  time.Time
	Text      []byte
}

func (v ID) GetTopic() string {
	return v.Topic
}

func (v ID) GetFrom() string {
	return v.From
}

func (v ID) GetTo() string {
	return v.To
}

func (v ID) GetType() MsgType {
	return v.Type
}

// OptRequest ui向node发送的操作请求
type OptRequest interface {
	OptType() int
	Response() chan interface{}
}

func ParseMsg(b []byte) Message {
	s := string(b)
	msgType := getMsgType(s)
	switch msgType {
	case TestMessageType:
		message := new(TestMessage)
		err := message.UnmarshalJSON(b)
		ErrPanic(err, Logger)
		return message
	case VoteBriefType:
		message := new(VoteBrief)
		err := message.UnmarshalJSON(b)
		ErrPanic(err, Logger)
		return message
	case VoteAcceptationType:
		message := new(VoteAcceptation)
		err := message.UnmarshalJSON(b)
		ErrPanic(err, Logger)
		return message
	case VoteDetailType:
		message := new(VoteDetail)
		err := message.UnmarshalJSON(b)
		ErrPanic(err, Logger)
		return message
	case VoteParamsType:
		message := new(VoteParams)
		err := message.UnmarshalJSON(b)
		ErrPanic(err, Logger)
		return message
	case VoteResultType:
		message := new(VoteResult)
		err := message.UnmarshalJSON(b)
		ErrPanic(err, Logger)
		return message
	case PolyResultType:
		message := new(PolyResult)
		err := message.UnmarshalJSON(b)
		ErrPanic(err, Logger)
		return message
	case PubKeyPartType:
		message := new(PubKeyPart)
		err := message.UnmarshalJSON(b)
		ErrPanic(err, Logger)
		return message
	case PubKeyType:
		message := new(PubKey)
		err := message.UnmarshalJSON(b)
		ErrPanic(err, Logger)
		return message
	case PlainTextPartType:
		message := new(PlainTextPart)
		err := message.UnmarshalJSON(b)
		ErrPanic(err, Logger)
		return message
	case CipherTextType:
		message := new(CipherText)
		err := message.UnmarshalJSON(b)
		ErrPanic(err, Logger)
		return message
	case IDType:
		message := new(ID)
		err := message.UnmarshalJSON(b)
		ErrPanic(err, Logger)
		return message
	}
	Logger.Infof("消息解析失败，未找到匹配的消息类型")
	return nil
}

func getMsgType(s string) MsgType {
	code, _ := strconv.Atoi(s[8:11])
	return MsgType(code)
}
