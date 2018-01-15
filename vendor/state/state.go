package state

type State int

const (
	NotInitiated State = iota
	Waiting
	Aborted
	Prepared
	Committed
)
