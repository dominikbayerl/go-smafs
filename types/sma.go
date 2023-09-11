package types

type FSResponse struct {
	Devices map[string]map[string][]FSEntry `json:"result"`
}

type FSEntry struct {
	Filename      string `json:"f,omitempty"`
	DirectoryName string `json:"d,omitempty"`
	Timestamp     uint64 `json:"tm"`
	Size          uint64 `json:"s,omitempty"`
}
