package agentdecision

// MaxBatch bounds how many proposals one request decides, or previews for
// deciding together. Each is decided and, when approved, executed in turn,
// so the bound is what keeps one request from running for minutes.
const MaxBatch = 50
