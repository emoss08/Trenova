import type { SelectOption } from "@trenova/shared/types/fields";
import type { z } from "zod";
import { emailProfileSchema } from "@trenova/shared/types/email";

export const emailProfileQueryKey = "email-profile-list";

export const emailProviderChoices: SelectOption[] = [
  { value: "Resend", label: "Resend" },
  { value: "Postmark", label: "Postmark" },
];

export const emailProfileStatusChoices: SelectOption[] = [
  { value: "Active", label: "Active", color: "var(--success)" },
  { value: "Inactive", label: "Inactive", color: "var(--danger)" },
];

export const emailPurposes = [
  "General",
  "Billing",
  "Reporting",
  "Operations",
  "Authentication",
  "Notifications",
] as const;

export type EmailPurpose = (typeof emailPurposes)[number];

export type EmailProfileFormValues = z.input<typeof emailProfileSchema>;

export const emailProfileDefaults: EmailProfileFormValues = {
  name: "",
  description: "",
  senderName: "",
  senderEmail: "",
  replyToEmail: "",
  provider: "Resend",
  status: "Active",
};
