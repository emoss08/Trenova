import type { GenericSelectOption } from "../types/fields";

export type SharedCommentType =
  | "Internal"
  | "Dispatch"
  | "DriverUpdate"
  | "PickupInstruction"
  | "DeliveryInstruction"
  | "StatusUpdate"
  | "Exception"
  | "CustomerUpdate"
  | "Appointment"
  | "Document"
  | "Billing"
  | "Compliance";

export type SharedCommentVisibility =
  | "Internal"
  | "Operations"
  | "Customer"
  | "Driver"
  | "Accounting";

export type SharedCommentPriority = "Low" | "Normal" | "High" | "Urgent";

export const commentTypeChoices = [
  { value: "Internal", label: "Internal", color: "var(--foreground-subtle)" },
  { value: "Dispatch", label: "Dispatch", color: "var(--info)" },
  { value: "DriverUpdate", label: "Driver update", color: "var(--accent-teal)" },
  { value: "PickupInstruction", label: "Pickup instruction", color: "var(--success)" },
  { value: "DeliveryInstruction", label: "Delivery instruction", color: "var(--success-foreground)" },
  { value: "StatusUpdate", label: "Status update", color: "var(--accent-indigo)" },
  { value: "Exception", label: "Exception", color: "var(--danger)" },
  { value: "CustomerUpdate", label: "Customer update", color: "var(--accent-violet)" },
  { value: "Appointment", label: "Appointment", color: "var(--warning)" },
  { value: "Document", label: "Document", color: "var(--foreground-muted)" },
  { value: "Billing", label: "Billing", color: "var(--accent-teal-on-subtle)" },
  { value: "Compliance", label: "Compliance", color: "var(--accent-rose)" },
] satisfies ReadonlyArray<GenericSelectOption<SharedCommentType>>;

export const commentVisibilityChoices = [
  { value: "Internal", label: "Internal", color: "var(--foreground-subtle)" },
  { value: "Operations", label: "Operations", color: "var(--info)" },
  { value: "Customer", label: "Customer", color: "var(--accent-violet)" },
  { value: "Driver", label: "Driver", color: "var(--accent-teal)" },
  { value: "Accounting", label: "Accounting", color: "var(--accent-teal-on-subtle)" },
] satisfies ReadonlyArray<GenericSelectOption<SharedCommentVisibility>>;

export const commentPriorityChoices = [
  { value: "Low", label: "Low", color: "var(--foreground-subtle)" },
  { value: "Normal", label: "Normal", color: "var(--info)" },
  { value: "High", label: "High", color: "var(--warning)" },
  { value: "Urgent", label: "Urgent", color: "var(--danger)" },
] satisfies ReadonlyArray<GenericSelectOption<SharedCommentPriority>>;

export function commentTypeLabel(value: string): string {
  return commentTypeChoices.find((choice) => choice.value === value)?.label ?? value;
}

export function commentTypeColor(value: string): string | undefined {
  return commentTypeChoices.find((choice) => choice.value === value)?.color;
}
