package mysql

const (
	Version = "5.7.25-Grant-DB"

	MaxPayloadLen = 1<<24 - 1
)

const (
	OKHeader          byte = 0x00
	ErrHeader         byte = 0xff
	EOFHeader         byte = 0xfe
	LocalInFileHeader byte = 0xfb
)

const (
	CmdSleep byte = iota
	CmdQuit
	CmdInitDB
	CmdQuery
)
