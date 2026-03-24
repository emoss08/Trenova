package system

type DatabaseSessionChain struct {
	BlockedPID              int64  `json:"blockedPid"                    bun:"blocked_pid"`
	BlockingPID             int64  `json:"blockingPid"                   bun:"blocking_pid"`
	DatabaseName            string `json:"databaseName"                  bun:"database_name"`
	BlockedState            string `json:"blockedState"                  bun:"blocked_state"`
	BlockingState           string `json:"blockingState"                 bun:"blocking_state"`
	BlockedWaitEventType    string `json:"blockedWaitEventType"          bun:"blocked_wait_event_type"`
	BlockedWaitEvent        string `json:"blockedWaitEvent"              bun:"blocked_wait_event"`
	BlockedApplicationName  string `json:"blockedApplicationName"        bun:"blocked_application_name"`
	BlockingApplicationName string `json:"blockingApplicationName"       bun:"blocking_application_name"`
	BlockedUser             string `json:"blockedUser"                   bun:"blocked_user"`
	BlockingUser            string `json:"blockingUser"                  bun:"blocking_user"`
	BlockedQueryPreview     string `json:"blockedQueryPreview"           bun:"blocked_query_preview"`
	BlockingQueryPreview    string `json:"blockingQueryPreview"          bun:"blocking_query_preview"`
	BlockedTransactionAgeS  int64  `json:"blockedTransactionAgeSeconds"  bun:"blocked_transaction_age_s"`
	BlockingTransactionAgeS int64  `json:"blockingTransactionAgeSeconds" bun:"blocking_transaction_age_s"`
	BlockedQueryAgeS        int64  `json:"blockedQueryAgeSeconds"        bun:"blocked_query_age_s"`
	BlockingQueryAgeS       int64  `json:"blockingQueryAgeSeconds"       bun:"blocking_query_age_s"`
}

type TerminateDatabaseSessionResult struct {
	PID        int64  `json:"pid"`
	Terminated bool   `json:"terminated"`
	Message    string `json:"message"`
}
