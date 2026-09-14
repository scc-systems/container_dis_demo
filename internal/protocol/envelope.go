package protocol

import "encoding/json"

const (
	TypeHello         = "hello"
	TypeWelcome       = "welcome"
	TypePing          = "ping"
	TypePong          = "pong"
	TypeSceneChange   = "scene_change"
	TypeTick          = "tick"
	TypeAdminStatus   = "admin_status"
	TypeNodeStats     = "node_stats"
	TypeVideoDuration = "video_duration"
)

type Envelope struct {
	Type       string          `json:"type"`
	Seq        uint64          `json:"seq,omitempty"`
	ServerTsMs int64           `json:"server_ts_ms,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
}

func Marshal(msgType string, seq uint64, serverTsMs int64, payload any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	env := Envelope{
		Type:       msgType,
		Seq:        seq,
		ServerTsMs: serverTsMs,
		Payload:    raw,
	}
	return json.Marshal(env)
}

func Unmarshal(data []byte) (Envelope, error) {
	var env Envelope
	err := json.Unmarshal(data, &env)
	return env, err
}

func UnmarshalPayload(env Envelope, into any) error {
	return json.Unmarshal(env.Payload, into)
}
