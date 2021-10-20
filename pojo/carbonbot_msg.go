package pojo

type CarbonbotMessage struct {
	Exchange    string `json:"exchange"`
	MarketType  string `json:"market_type"`
	MessageType string `json:"message_type"`
	ReceivedAt  uint64 `json:"received_at"`
	Json        string `json:"json"`
}
