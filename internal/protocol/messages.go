package protocol

type HelloPayload struct {
	NodeID     int    `json:"node_id"`
	ClientTsMs int64  `json:"client_ts_ms"`
	Screen     Screen `json:"screen"`
	UserAgent  string `json:"user_agent,omitempty"`
}

type Screen struct {
	W int `json:"w"`
	H int `json:"h"`
}

type AssignedTile struct {
	Col  int   `json:"col"`
	Rows []int `json:"rows"`
}

type WelcomePayload struct {
	NodeID        int          `json:"node_id"`
	AssignedTile  AssignedTile `json:"assigned_tile"`
	CurrentScene  string       `json:"current_scene"`
	SceneEpochMs  int64        `json:"scene_epoch_ms"`
	ServerTimeMs  int64        `json:"server_time_ms"`
	TickHz        int          `json:"tick_hz"`
	WallWidth     int          `json:"wall_width"`
	WallHeight    int          `json:"wall_height"`
	PerNodeWidth  int          `json:"per_node_width"`
	PerNodeHeight int          `json:"per_node_height"`
	NodeCount     int          `json:"node_count"`
	Seed          uint32       `json:"seed"`
}

type PingPayload struct {
	Nonce string `json:"nonce"`
}

type PongPayload struct {
	Nonce          string `json:"nonce"`
	ServerTsMsEcho int64  `json:"server_ts_ms_echo"`
	ClientTsMs     int64  `json:"client_ts_ms"`
}

type SceneChangePayload struct {
	Scene   string         `json:"scene"`
	EpochMs int64          `json:"epoch_ms"`
	Params  map[string]any `json:"params,omitempty"`
}

type TickPayload struct {
	Scene   string         `json:"scene"`
	TMs     int64          `json:"t_ms"`
	Camera  *CameraTick    `json:"camera,omitempty"`
	Seed    uint32         `json:"seed,omitempty"`
	Spawner *SpawnerTick   `json:"spawner,omitempty"`
	FFT     []float32      `json:"fft,omitempty"`
	RadarDeg float32       `json:"radar_deg,omitempty"`
	Blips   []Blip         `json:"blips,omitempty"`
}

type CameraTick struct {
	OrbitDeg float32 `json:"orbit_deg"`
}

type SpawnerTick struct {
	Rate    float32 `json:"rate"`
	GustDeg float32 `json:"gust_deg"`
}

type Blip struct {
	R     float32 `json:"r"`
	Theta float32 `json:"theta"`
	ID    string  `json:"id"`
}

type AdminNodeStatus struct {
	ID            int     `json:"id"`
	Connected     bool    `json:"connected"`
	RTTMedianMs   float64 `json:"rtt_med_ms"`
	RTTP95Ms      float64 `json:"rtt_p95_ms"`
	OffsetMs      float64 `json:"offset_ms"`
	LastPongMsAgo int64   `json:"last_pong_ms_ago"`
	Drops         uint64  `json:"drops"`
	ConnectedAtMs int64   `json:"connected_at_ms,omitempty"`
}

type AdminStatusPayload struct {
	Scene      string            `json:"scene"`
	EpochMs    int64             `json:"epoch_ms"`
	NowMs      int64             `json:"now_ms"`
	AutoRotate bool              `json:"auto_rotate"`
	Nodes      []AdminNodeStatus `json:"nodes"`
}

type NodeStatsPayload struct {
	RTTMedianMs float64 `json:"rtt_med_ms"`
	RTTP95Ms    float64 `json:"rtt_p95_ms"`
	OffsetMs    float64 `json:"offset_ms"`
	Drops       uint64  `json:"drops"`
}
