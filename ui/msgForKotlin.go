package ui

import "time"

type msgType int

const (
	MsgNewVoteType msgType = iota
	MsgVoteDetailsType
	MsgPartiesNumType
	MsgResultType
	MsgGotPKType
)

type msg4Kot interface {
	GetType() msgType
	MarshalJSON() ([]byte, error)
	UnmarshalJSON([]byte) error
}
type MsgResult struct {
	MType     msgType
	Result    map[string]int
	IsCorrect bool
}

func (m MsgResult) GetType() msgType { return m.MType }

type MsgNewVote struct {
	MType    msgType
	Starter  string
	Title    string
	Brief    string
	VoteType string
	Start    time.Time
	Deadline time.Time
}

func (m MsgNewVote) GetType() msgType { return m.MType }

type MsgVoteDetails struct {
	MType      msgType
	Title      string
	Brief      string
	Candidates []string
}

func (m MsgVoteDetails) GetType() msgType { return m.MType }

type MsgPartiesNum struct {
	MType msgType
	Num   int
}

func (m MsgPartiesNum) GetType() msgType { return m.MType }

type MsgGotPK struct {
	MType msgType
}

func (m MsgGotPK) GetType() msgType { return m.MType }
