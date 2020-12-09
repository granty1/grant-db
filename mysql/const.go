package mysql

const (
	Version = "5.7.25-Grant-DB"

	MaxPayloadLen = 1<<24 - 1
)

const (
	CmdSleep byte = iota
	CmdQuit
	CmdInitDB
	CmdQuery
)
