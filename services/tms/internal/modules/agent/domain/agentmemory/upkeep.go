package agentmemory

const (
	secondsPerDay = int64(24 * 60 * 60)
	// UnusedAfterDays is how long a memory no turn has used is kept before a
	// person is asked whether it is worth keeping.
	UnusedAfterDays = 30
	// AgingAfterDays is how long a fact with no end date stands before a
	// person is asked whether it is still true.
	AgingAfterDays = 180
	// OfferLifetimeDays is how long an offered memory waits for a person. One
	// nobody decided by then lapses: it is set aside as not kept, can still be
	// accepted where it was offered, and stops crowding the offers that are
	// new.
	OfferLifetimeDays = 30
)

// OffersLapseBefore is the time an offer must have been made by to lapse now.
func OffersLapseBefore(now int64) int64 {
	return now - OfferLifetimeDays*secondsPerDay
}

type MemoryUpkeepReason string

const (
	MemoryUpkeepNeverUsed = MemoryUpkeepReason("NeverUsed")
	MemoryUpkeepAging     = MemoryUpkeepReason("Aging")
)

type MemoryUpkeepWindow struct {
	UnusedBefore int64
	AgingBefore  int64
}

func UpkeepWindow(now int64) MemoryUpkeepWindow {
	return MemoryUpkeepWindow{
		UnusedBefore: now - UnusedAfterDays*secondsPerDay,
		AgingBefore:  now - AgingAfterDays*secondsPerDay,
	}
}

// VouchedAt is when a person last stood behind the memory: when they said it
// still holds, or else when it was recorded.
func (m *Memory) VouchedAt() int64 {
	if m.ConfirmedAt != nil && *m.ConfirmedAt > m.CreatedAt {
		return *m.ConfirmedAt
	}

	return m.CreatedAt
}

// UpkeepReason says why a shared, active memory is worth another look, or
// nothing when it is not. An aging fact is asked about before an unused one:
// whether it is still true matters more than whether it is read.
func (m *Memory) UpkeepReason(window MemoryUpkeepWindow) MemoryUpkeepReason {
	if m.Status != MemoryStatusActive || m.Scope == MemoryScopeUser {
		return ""
	}

	vouched := m.VouchedAt()
	switch {
	case m.Kind == MemoryKindFact && m.ExpiresAt == nil && vouched < window.AgingBefore:
		return MemoryUpkeepAging
	case m.UseCount == 0 && vouched < window.UnusedBefore:
		return MemoryUpkeepNeverUsed
	default:
		return ""
	}
}
