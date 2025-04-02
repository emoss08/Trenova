export enum AccessorialChargeMethod {
  Flat = "Flat",
  Distance = "Distance",
  Percentage = "Percentage",
}

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

export enum PaymentTerm {
  Net15 = "Net15",
  Net30 = "Net30",
  Net45 = "Net45",
  Net60 = "Net60",
  Net90 = "Net90",
  DueOnReceipt = "DueOnReceipt",
}

export enum TransferSchedule {
  Continuous = "Continuous",
  Hourly = "Hourly",
  Daily = "Daily",
  Weekly = "Weekly",
}

export enum DocumentClassification {
  Public = "Public",
  Private = "Private",
  Sensitive = "Sensitive",
  Regulatory = "Regulatory",
}

export enum DocumentCategory {
  Shipment = "Shipment",
  Worker = "Worker",
  Regulatory = "Regulatory",
  Profile = "Profile",
  Branding = "Branding",
  Invoice = "Invoice",
  Contract = "Contract",
  Other = "Other",
}
