package messagetype

type MessageType int

const (
	DATA MessageType = iota
	INFO
	LOG_MESSAGE
)

// String method to convert enum values to string
func (s MessageType) String() string {
	switch s {
	case DATA:
		return "DATA"
	case INFO:
		return "INFO"
	case LOG_MESSAGE:
		return "LOG_MESSAGE"
	default:
		return "UNKNOWN"
	}
}
