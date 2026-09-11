package scene

const (
	SceneIdle          = "idle"
	SceneThreeUnified  = "three-unified"
	SceneSolarPanorama = "solar-panorama"
	SceneWarpJump      = "warp-jump"
	SceneVideo         = "video"
	SceneImage         = "image"
)

// All lists the scenes eligible for auto-rotation.
// Idle, video, and image are operator-triggered only.
var All = []string{SceneThreeUnified, SceneSolarPanorama, SceneWarpJump}

// Valid returns true for any scene that Switch will accept.
func Valid(s string) bool {
	if s == SceneVideo || s == SceneImage || s == SceneIdle {
		return true
	}
	for _, v := range All {
		if v == s {
			return true
		}
	}
	return false
}
