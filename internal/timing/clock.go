package timing

import "time"

func EpochNowMs() int64 {
	return time.Now().UnixMilli()
}
