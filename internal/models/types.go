package models

type SRSCallback struct {
	Action    string `json:"action"`
	App       string `json:"app"`
	Stream    string `json:"stream"`
	Param     string `json:"param"`
	// Cambio: ampliar campos de callback para on_play/on_stop (Firma: Cursor)
	ClientID  string `json:"client_id"`
	IP        string `json:"ip"`
	Vhost     string `json:"vhost"`
	StreamURL string `json:"stream_url"`
	StreamID  string `json:"stream_id"`
	PageURL   string `json:"pageUrl"`
	Opaque    string `json:"opaque"`
	RequestID string `json:"request_id"`
}

type ServerStats struct {
	Uptime       int64  `json:"uptime"`
	Connections  int    `json:"connections"`
	TotalStreams int    `json:"total_streams"`
	Version      string `json:"version"`
}

type StreamInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	App        string `json:"app"`
	Clients    int    `json:"clients"`
	RecvKbps   int    `json:"recv_kbps"`
	SendKbps   int    `json:"send_kbps"`
	IsPublish  bool   `json:"is_publish"`
	VideoCodec string `json:"video_codec"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
}

type ResourceStats struct {
	CPU    float64 `json:"cpu"`
	Memory int64   `json:"memory"`
}

type SRSStats struct {
	Server    ServerStats   `json:"server"`
	Streams   []StreamInfo  `json:"streams"`
	Resources ResourceStats `json:"resources"`
}

type ClientInfo struct {
	ID        string `json:"id"`
	IP        string `json:"ip"`
	Type      string `json:"type"`
	Stream    string `json:"stream"`
	App       string `json:"app"`
	Alive     int64  `json:"alive"`
	SendBytes int64  `json:"send_bytes"`
	RecvBytes int64  `json:"recv_bytes"`
}

type PerformanceStats struct {
	CPU          float64 `json:"cpu"`
	Memory       int64   `json:"memory_mb"`
	TotalPackets int64   `json:"total_packets"`
	TotalFrames  int64   `json:"total_frames"`
	FreeObjects  int     `json:"free_objects"`
	Connections  int     `json:"connections"`
}

type ServerSummary struct {
	Version      string `json:"version"`
	PID          int    `json:"pid"`
	Uptime       int64  `json:"uptime"`
	Publishers   int    `json:"publishers"`
	Players      int    `json:"players"`
	TotalClients int    `json:"total_clients"`
}