package tendernegotiation

// MaxDeclineReasonLength bounds carrier-supplied text before it travels into
// workflow signals, events, and notifications; the client caps at the same
// length.
const MaxDeclineReasonLength = 500
