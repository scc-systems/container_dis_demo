package feeds

import "math"

// GenerateFFT synthesises a deterministic 256-bin "FFT" from t_ms.
// It's a sum of moving sines + a slow bass thump; values in [0,1].
func GenerateFFT(tMs int64, bins int) []float32 {
	if bins <= 0 {
		bins = 256
	}
	out := make([]float32, bins)
	t := float64(tMs) / 1000.0
	for i := 0; i < bins; i++ {
		f := float64(i) / float64(bins) // 0..1 across the spectrum
		// Bass thump around f=0.05 modulated at ~2 Hz.
		bass := 0.6 * math.Exp(-math.Pow((f-0.05)/0.03, 2)) * (0.5 + 0.5*math.Sin(2*math.Pi*2*t))
		// Mid components sweeping in frequency.
		mid := 0.35 * math.Exp(-math.Pow((f-(0.25+0.15*math.Sin(0.4*t)))/0.05, 2)) * (0.6 + 0.4*math.Sin(2*math.Pi*7*t))
		// A drifting high spike.
		hi := 0.28 * math.Exp(-math.Pow((f-(0.7+0.1*math.Sin(0.13*t+1.2)))/0.02, 2)) * (0.5 + 0.5*math.Sin(2*math.Pi*11*t+0.7))
		// Base noise floor deterministic on (i, floor(t*8)).
		floor := 0.05 + 0.05*math.Sin(float64(i)*0.37+t*3.1)
		v := bass + mid + hi + floor
		if v < 0 {
			v = 0
		}
		if v > 1 {
			v = 1
		}
		out[i] = float32(v)
	}
	return out
}
