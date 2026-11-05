package workeremploymentservice

func (s *Service) SetCollaborators(
	workers WorkerUpdater,
	policies PTOPolicyManager,
	ledger PTOSettler,
	pay PayAssignmentManager,
) {
	s.workers = workers
	s.ptoPolicies = policies
	s.ptoLedger = ledger
	s.driverPay = pay
}
