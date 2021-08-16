package netop

import (
	"fmt"
	"math"
	"time"
)

// Progress download progress
type Progress struct {
	Total     int64
	Completed int64
	Speed     int64
	Elapsed   time.Duration
	Remain    time.Duration
}

func (p Progress) String() string {
	var totalString string
	if p.Total > 1024*1024*1024 {
		totalString = fmt.Sprintf("%.2fGB", float64(p.Total)/float64(1024*1024*1024))
	} else if p.Total > 1024*1024 {
		totalString = fmt.Sprintf("%.2fMB", float64(p.Total)/float64(1024*1024))
	} else if p.Total > 1024 {
		totalString = fmt.Sprintf("%.2fKB", float64(p.Total)/float64(1024))
	} else {
		totalString = fmt.Sprintf("%.2fB", float64(p.Total))
	}

	var speedString string
	if p.Speed > 1024*1024*1024 {
		speedString = fmt.Sprintf("%.2fGB/s", float64(p.Speed)/float64(1024*1024*1024))
	} else if p.Speed > 1024*1024 {
		speedString = fmt.Sprintf("%.2fMB/s", float64(p.Speed)/float64(1024*1024))
	} else if p.Speed > 1024 {
		speedString = fmt.Sprintf("%.2fKB/s", float64(p.Speed)/float64(1024))
	} else {
		speedString = fmt.Sprintf("%.2fB/s", float64(p.Speed))
	}

	remain := "-"
	if int64(p.Remain) != math.MaxInt64 {
		remain = p.Remain.String()
	}

	return fmt.Sprintf("progress: %.2f%%  total: %s  speed: %s  elapsed: %s remain: %s",
		float64(p.Completed)/float64(p.Total)*100,
		totalString,
		speedString,
		p.Elapsed.String(),
		remain,
	)
}
