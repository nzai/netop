package netop

import (
	"math"
	"sync"
	"time"
)

const (
	CalculateSpeedInterval = time.Second
)

type SpeedWatch struct {
	total        int64
	complete     int64
	lastComplete int64
	start        time.Time
	ticker       *time.Ticker
	mutex        *sync.RWMutex
	speed        int64
}

func NewSpeedWatch(total int64) *SpeedWatch {
	sw := &SpeedWatch{
		total:        total,
		complete:     0,
		lastComplete: 0,
		start:        time.Now(),
		ticker:       time.NewTicker(CalculateSpeedInterval),
		mutex:        new(sync.RWMutex),
	}

	go func() {
		for range sw.ticker.C {
			sw.calculateSpeed()
		}
	}()

	return sw
}

func (s *SpeedWatch) Do(work int64) {
	if work <= 0 {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.complete += work
	if s.complete > s.total {
		s.complete = s.total
	}
}

func (s *SpeedWatch) calculateSpeed() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.speed = (s.complete - s.lastComplete) / int64(CalculateSpeedInterval.Seconds())
	s.lastComplete = s.complete
}

func (s *SpeedWatch) Progress() *Progress {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	remain := time.Duration(math.MaxInt64)
	if s.speed > 0 {
		remain = time.Second * time.Duration((s.total-s.complete)/s.speed)
		if remain < 0 {
			remain = 0
		}
	}

	return &Progress{
		Total:     s.total,
		Completed: s.complete,
		Speed:     s.speed,
		Elapsed:   time.Since(s.start),
		Remain:    remain,
	}
}

func (s *SpeedWatch) Stop() {
	if s.ticker == nil {
		return
	}

	s.ticker.Stop()
}
