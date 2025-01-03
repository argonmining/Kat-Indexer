package models

type OperationsResponse struct {
	Message string      `json:"message"`
	Result  []Operation `json:"result"`
	HasMore bool        `json:"hasMore"`
}

type Operation struct {
	P           string `json:"p"`
	Op          string `json:"op"`
	Tick        string `json:"tick"`
	Amt         string `json:"amt"`
	From        string `json:"from"`
	To          string `json:"to"`
	OpScore     string `json:"opScore"`
	HashRev     string `json:"hashRev"`
	FeeRev      string `json:"feeRev"`
	FeeLeast    string `json:"feeLeast"`
	TxAccept    string `json:"txAccept"`
	BlockAccept string `json:"blockAccept"`
	OpAccept    string `json:"opAccept"`
	OpError     string `json:"opError"`
	Checkpoint  string `json:"checkpoint"`
	MtsAdd      string `json:"mtsAdd"`
	MtsMod      string `json:"mtsMod"`
}
