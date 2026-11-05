package fuelpurchaseservice

func (s *Service) SetClock(now func() int64) {
	s.now = now
}
