package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr            string
	BasePath        string
	TickHz          int
	PingInterval    time.Duration
	InitialScene    string
	AutoRotate      time.Duration
	SceneSwitchLead time.Duration
	MediaDir        string
	MaxUploadBytes  int64

	WallWidth      int
	WallHeight     int
	NodeCount      int
	PerNodeWidth   int
	PerNodeHeight  int
	TileCols       int
	TileRows       int
	TileWidth      int
	TileHeight     int
}

func Load() Config {
	c := Config{
		Addr:            envStr("WALLSYNC_ADDR", ":8080"),
		BasePath:        normalizePath(envStr("WALLSYNC_BASE_PATH", "")),
		TickHz:          envInt("WALLSYNC_TICK_HZ", 30),
		PingInterval:    envDur("WALLSYNC_PING_MS", 1000) * time.Millisecond,
		InitialScene:    envStr("WALLSYNC_INITIAL_SCENE", "idle"),
		AutoRotate:      time.Duration(envInt("WALLSYNC_AUTOROTATE_S", 20)) * time.Second,
		SceneSwitchLead: 2000 * time.Millisecond,
		MediaDir:        envStr("WALLSYNC_MEDIA_DIR", "./data/videos"),
		MaxUploadBytes:  int64(envInt("WALLSYNC_MAX_UPLOAD_MB", 500)) * 1024 * 1024,
		NodeCount:       5,
		TileCols:        3,
		TileRows:        3,
		TileWidth:       1920,
		TileHeight:      1080,
	}
	c.PerNodeWidth = c.TileCols * c.TileWidth
	c.PerNodeHeight = c.TileRows * c.TileHeight
	c.WallWidth = c.NodeCount * c.PerNodeWidth
	c.WallHeight = c.PerNodeHeight
	return c
}

// normalizePath strips trailing slashes and ensures a leading slash for non-empty paths.
func normalizePath(p string) string {
	for len(p) > 0 && p[len(p)-1] == '/' {
		p = p[:len(p)-1]
	}
	if p != "" && p[0] != '/' {
		p = "/" + p
	}
	return p
}

func envStr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envDur(key string, fallbackMs int) time.Duration {
	return time.Duration(envInt(key, fallbackMs))
}
