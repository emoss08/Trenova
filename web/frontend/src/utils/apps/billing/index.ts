import type { IChoiceProps, TChoiceProps } from "@/types";

/** Type for fuel method choices */
export type FuelMethodChoicesProps = "Distance" | "Flat" | "Percentage";

export const fuelMethodChoices = [
  { value: "Distance", label: "Distance" },
  { value: "Flat", label: "Flat" },
  { value: "Percentage", label: "Percentage" },
] satisfies ReadonlyArray<IChoiceProps<FuelMethodChoicesProps>>;

/** Type for Auto Billing Criteria Choices */
export type AutoBillingCriteriaChoicesProps =
  | "Delivered"
  | "TransferredToBilling"
  | "MarkedReadyToBill";

export const autoBillingCriteriaChoices = [
  { value: "Delivered", label: "Auto Bill when shipment is delivered" },
  {
    value: "TransferredToBilling",
    label: "Auto Bill when order are transferred to billing",
  },
  {
    value: "MarkedReadyToBill",
    label: "Auto Bill when order is marked ready to bill in Billing Queue",
  },
] satisfies ReadonlyArray<IChoiceProps<AutoBillingCriteriaChoicesProps>>;

/** Type for order transfer criteria */
export type ShipmentTransferCriteriaChoicesProps =
  | "ReadyAndCompleted"
  | "Completed"
  | "ReadyToBill";

export const shipmentTransferCriteriaChoices = [
  { value: "ReadyAndCompleted", label: "Ready to bill & Completed" },
  { value: "Completed", label: "Completed" },
  { value: "ReadyToBill", label: "Ready to bill" },
] satisfies ReadonlyArray<IChoiceProps<ShipmentTransferCriteriaChoicesProps>>;

/** Type for Bill Type choices */
export type billTypeChoicesProps =
  | "INVOICE"
  | "CREDIT"
  | "DEBIT"
  | "PREPAID"
  | "OTHER";

export const billTypeChoices: TChoiceProps[] = [
  { value: "INVOICE", label: "Invoice" },
  { value: "CREDIT", label: "Credit" },
  { value: "DEBIT", label: "Debit" },
  { value: "PREPAID", label: "Prepaid" },
  { value: "OTHER", label: "Other" },
];

/** Type for Billing Exception choices */
export type billingExceptionChoicesProps =
  | "PAPERWORK"
  | "CHARGE"
  | "CREDIT"
  | "DEBIT"
  | "OTHER";

export const billingExceptionChoices: TChoiceProps[] = [
  { value: "PAPERWORK", label: "Paperwork" },
  { value: "CHARGE", label: "Charge" },
  { value: "CREDIT", label: "Credit" },
  { value: "DEBIT", label: "Debit" },
  { value: "OTHER", label: "OTHER" },
];
