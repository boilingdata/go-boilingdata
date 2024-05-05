package messages

type Payload struct {
	MessageType string `json:"messageType"`
	SQL         string `json:"sql"`
	RequestID   string `json:"requestId"`
}

type Response struct {
	MessageType       string                   `json:"messageType"`
	RequestID         string                   `json:"requestId"`
	BatchSerial       int                      `json:"batchSerial"`
	TotalBatches      int                      `json:"totalBatches"`
	SplitSerial       int                      `json:"splitSerial"`
	TotalSplitSerials int                      `json:"totalSplitSerials"`
	CacheInfo         string                   `json:"cacheInfo"`
	SubBatchSerial    int                      `json:"subBatchSerial"`
	TotalSubBatches   int                      `json:"totalSubBatches"`
	Data              []map[string]interface{} `json:"data"`
	Keys              []string                 `json:"-"`
}

// Define structs to represent the JSON payload
type Tag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func GetPayLoad() Payload {
	return Payload{
		MessageType: "SQL_QUERY",
		SQL:         "",
		RequestID:   "",
	}
}

/// Responses

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

type LogMessage struct {
	MessageType string `json:"messageType"`
	LogLevel    string `json:"logLevel"`
	RequestID   string `json:"requestId"`
	LogMessage  string `json:"logMessage"`
}
