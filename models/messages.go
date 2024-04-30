package models

type Payload struct {
	MessageType string `json:"messageType"`
	SQL         string `json:"sql"`
	RequestID   string `json:"requestId"`
	ReadCache   string `json:"readCache"`
	Tags        []Tag  `json:"tags"`
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
		ReadCache:   "NONE",
		Tags: []Tag{
			{
				Name:  "CostCenter",
				Value: "930",
			},
			{
				Name:  "ProjectId",
				Value: "Top secret Area 53",
			},
		},
	}
}
