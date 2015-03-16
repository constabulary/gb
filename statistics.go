package gb

// build statistics

import (
	"fmt"
	"sync"
	"time"
)

// Statistics records the various Durations
type Statistics struct {
	sync.Mutex
	stats map[string]time.Duration
}

func (s *Statistics) Record(name string, d time.Duration) {
	s.Lock()
	defer s.Unlock()
	if s.stats == nil {
		s.stats = make(map[string]time.Duration)
	}
	acc := s.stats[name]
	acc += d
	s.stats[name] = acc
}

func (s *Statistics) Total() time.Duration {
	s.Lock()
	defer s.Unlock()
	var d time.Duration
	for _, v := range s.stats {
		d += v
	}
	return d
}

func (s *Statistics) String() string {
	s.Lock()
	defer s.Unlock()
	return fmt.Sprintf("%v", s.stats)
}
