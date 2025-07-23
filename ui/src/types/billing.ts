/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

export enum AccessorialChargeMethod {
  Flat = "Flat",
  Distance = "Distance",
  Percentage = "Percentage",
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

export enum BillingCycleType {
  Immediate = "Immediate",
  Daily = "Daily",
  Weekly = "Weekly",
  Monthly = "Monthly",
  Quarterly = "Quarterly",
}
