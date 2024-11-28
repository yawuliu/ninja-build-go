package main

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

func (this *SlidingRateInfo) UpdateRate(update_hint int, time_millis int64) {
	if update_hint == this.last_update_ {
		return
	}

	this.last_update_ = update_hint

	if this.times_.size() == this.N {
		this.times_.pop()
	}
	this.times_.push(time_millis)

	if this.times_.back() != this.times_.front() {
		this.rate_ = this.times_.size() / ((this.times_.back() - this.times_.front()) / 1e3)
	}
}

func Statusfactory(config *BuildConfig) Status {
	return NewStatusPrinter(config)
}
