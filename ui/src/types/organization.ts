export enum OrganizationType {
  Carrier = "Carrier",
  Brokerage = "Brokerage",
  BrokerageCarrier = "Brokerage & Carrier",
}

export type BusinessUnitStatus =
  | "Active"
  | "Inactive"
  | "Pending"
  | "Suspended";
