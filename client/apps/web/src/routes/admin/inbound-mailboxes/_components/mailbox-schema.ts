import type {
  InboundMailbox,
  InboundMailboxInput,
  InboundMailboxStatus,
  InboundProvider,
  InboundReviewPolicy,
} from "@/lib/graphql/inbox";
import { z } from "zod";

/** The server's floor: below it a classifier is guessing. */
export const MIN_CONFIDENCE_FLOOR = 0.5;

export const PROVIDERS = ["Resend", "Postmark"] as const satisfies readonly InboundProvider[];
export const REVIEW_POLICIES = [
  "AlwaysReview",
  "ReviewBelowConfidence",
  "AutoHandle",
] as const satisfies readonly InboundReviewPolicy[];
export const MAILBOX_STATUSES = [
  "Active",
  "Inactive",
] as const satisfies readonly InboundMailboxStatus[];

export const mailboxFormSchema = z
  .object({
    name: z.string().trim().min(1, { error: "Name is required" }).max(100),
    address: z.email({ error: "Address must be an email address" }).max(255),
    provider: z.enum(PROVIDERS),
    purpose: z.string().max(255),
    reviewPolicy: z.enum(REVIEW_POLICIES),
    minConfidence: z.number().min(0).max(1),
    status: z.enum(MAILBOX_STATUSES),
  })
  .refine(
    (value) =>
      value.reviewPolicy !== "ReviewBelowConfidence" || value.minConfidence >= MIN_CONFIDENCE_FLOOR,
    {
      error: `A confidence bar below ${MIN_CONFIDENCE_FLOOR} is auto-handling with a number in front of it`,
      path: ["minConfidence"],
    },
  );

export type MailboxFormValues = z.infer<typeof mailboxFormSchema>;

export function newMailboxDefaults(): MailboxFormValues {
  return {
    name: "",
    address: "",
    provider: "Resend",
    purpose: "",
    reviewPolicy: "AlwaysReview",
    minConfidence: 0.8,
    status: "Active",
  };
}

export function mailboxFormValues(mailbox: InboundMailbox): MailboxFormValues {
  return {
    name: mailbox.name,
    address: mailbox.address,
    provider: mailbox.provider,
    purpose: mailbox.purpose,
    reviewPolicy: mailbox.reviewPolicy,
    minConfidence: mailbox.minConfidence,
    status: mailbox.status,
  };
}

export function mailboxInput(values: MailboxFormValues): InboundMailboxInput {
  const purpose = values.purpose.trim();

  return {
    name: values.name.trim(),
    address: values.address.trim(),
    provider: values.provider,
    purpose: purpose === "" ? null : purpose,
    reviewPolicy: values.reviewPolicy,
    minConfidence: values.minConfidence,
    status: values.status,
  };
}
