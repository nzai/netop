package netop

import (
	"io"
	"sync"
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

// ProgressReadCloser read closer with progress event
type ProgressReadCloser struct {
	rc                io.ReadCloser
	total             int64
	ch                chan<- *Progress
	interval          time.Duration
	intervalOnce      *sync.Once
	intervalTicker    *time.Ticker
	startReadAt       time.Time
	lastNotifyAt      time.Time
	lastCompleted     int64
	intervalCompleted int64
}

// NewProgressReadCloser create new read closer with progress event
func NewProgressReadCloser(rc io.ReadCloser, total int64, ch chan<- *Progress, interval time.Duration) *ProgressReadCloser {
	if interval <= 0 {
		panic("invalid progress interval(must > 0)")
	}

	return &ProgressReadCloser{
		rc:           rc,
		total:        total,
		ch:           ch,
		interval:     interval,
		intervalOnce: new(sync.Once),
	}
}

func (s *ProgressReadCloser) Read(p []byte) (int, error) {
	s.intervalOnce.Do(func() {
		s.startReadAt = time.Now()
		s.lastNotifyAt = s.startReadAt
		s.intervalTicker = time.NewTicker(s.interval)
	})

	read, err := s.rc.Read(p)
	s.intervalCompleted += int64(read)
	select {
	case now := <-s.intervalTicker.C:
		s.notify(now, read)
	default:
	}

	return read, err
}

func (s *ProgressReadCloser) notify(now time.Time, read int) {
	interval := now.Sub(s.lastNotifyAt)
	if interval < s.interval {
		return
	}

	intervalSeconds := int64(interval.Seconds())
	if intervalSeconds == 0 {
		return
	}

	speed := s.intervalCompleted / intervalSeconds
	var remain time.Duration
	if speed != 0 {
		remain = time.Second * time.Duration((s.total-s.lastCompleted-s.intervalCompleted)/speed)
	}

	s.ch <- &Progress{
		Total:     s.total,
		Completed: s.lastCompleted + s.intervalCompleted,
		Speed:     speed,
		Elapsed:   now.Sub(s.startReadAt),
		Remain:    remain,
	}
	s.lastNotifyAt = now
	s.lastCompleted += s.intervalCompleted
	s.intervalCompleted = 0
}
