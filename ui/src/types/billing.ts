export enum TransferCriteria {
  ReadyAndCompleted = "ReadyAndCompleted",
  Completed = "Completed",
  ReadyToBill = "ReadyToBill",
  DocumentsAttached = "DocumentsAttached",
  PODReceived = "PODReceived",
}

export enum AutoBillCriteria {
  Delivered = "Delivered",
  Transferred = "Transferred",
  MarkedReadyToBill = "MarkedReadyToBill",
  PODReceived = "PODReceived",
  DocumentsVerified = "DocumentsVerified",
}

export enum BillingExceptionHandling {
  Queue = "Queue",
  Notify = "Notify",
  AutoResolve = "AutoResolve",
  Reject = "Reject",
}
