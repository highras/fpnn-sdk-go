package fpnn

const (
	FlagMsgpack = 0x80
	FlagJson    = 0x40
	FlagZip     = 0x20
	FlagEncrypt = 0x10
)

const (
	PackageTypeMsgpack = 0
	PackageTypeJson    = 1
)

const (
	MessageTypeOneWay = 0
	MessageTypeTwoWay = 1
	MessageTypeAnswer = 2
)

const (
	ProtoVersion = 1
)

const (
	MagicFPNN = "FPNN"
	MagicHTTP = "HTTP"
)