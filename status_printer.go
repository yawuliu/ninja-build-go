package main

import "time"

type SlidingRateInfo struct {
	rate_        float64
	N            int
	times_       []float64
	last_update_ int
}

func NewSlidingRateInfo(n int) *SlidingRateInfo {
	ret := SlidingRateInfo{}
	ret.rate_ = -1
	ret.N = n
	ret.last_update_ = -1
	return &ret
}

func (this *SlidingRateInfo) rate() float64 {
	return this.rate_
}

func (s *SlidingRateInfo) UpdateRate(update_hint int, time_millis int64) {
	if update_hint == s.last_update_ {
		return
	}
	s.last_update_ = update_hint

	if len(s.times_) == s.N {
		// 移除最旧的时间
		s.times_ = s.times_[1:]
	}
	// 添加新的时间
	s.times_ = append(s.times_, float64(time_millis))

	// 计算速率
	if len(s.times_) > 1 {
		interval := time.Duration(s.times_[len(s.times_)-1]-s.times_[0]) * time.Millisecond
		s.rate_ = float64(len(s.times_)) / interval.Seconds()
	} else {
		s.rate_ = -1
	}
}

func Statusfactory(config *BuildConfig) Status {
	return NewStatusPrinter(config)
}
