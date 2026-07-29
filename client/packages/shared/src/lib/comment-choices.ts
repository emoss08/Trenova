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
  { value: "Internal", label: "Internal", color: "#6b7280" },
  { value: "Dispatch", label: "Dispatch", color: "#3b82f6" },
  { value: "DriverUpdate", label: "Driver Update", color: "#0891b2" },
  { value: "PickupInstruction", label: "Pickup Instruction", color: "#16a34a" },
  { value: "DeliveryInstruction", label: "Delivery Instruction", color: "#15803d" },
  { value: "StatusUpdate", label: "Status Update", color: "#6366f1" },
  { value: "Exception", label: "Exception", color: "#dc2626" },
  { value: "CustomerUpdate", label: "Customer Update", color: "#a855f7" },
  { value: "Appointment", label: "Appointment", color: "#f59e0b" },
  { value: "Document", label: "Document", color: "#64748b" },
  { value: "Billing", label: "Billing", color: "#0d9488" },
  { value: "Compliance", label: "Compliance", color: "#db2777" },
] satisfies ReadonlyArray<GenericSelectOption<SharedCommentType>>;

export const commentVisibilityChoices = [
  { value: "Internal", label: "Internal", color: "#6b7280" },
  { value: "Operations", label: "Operations", color: "#3b82f6" },
  { value: "Customer", label: "Customer", color: "#a855f7" },
  { value: "Driver", label: "Driver", color: "#0891b2" },
  { value: "Accounting", label: "Accounting", color: "#0d9488" },
] satisfies ReadonlyArray<GenericSelectOption<SharedCommentVisibility>>;

export const commentPriorityChoices = [
  { value: "Low", label: "Low", color: "#9ca3af" },
  { value: "Normal", label: "Normal", color: "#3b82f6" },
  { value: "High", label: "High", color: "#f59e0b" },
  { value: "Urgent", label: "Urgent", color: "#dc2626" },
] satisfies ReadonlyArray<GenericSelectOption<SharedCommentPriority>>;

export function commentTypeLabel(value: string): string {
  return commentTypeChoices.find((choice) => choice.value === value)?.label ?? value;
}

export function commentTypeColor(value: string): string | undefined {
  return commentTypeChoices.find((choice) => choice.value === value)?.color;
}
