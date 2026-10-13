import type { CheckCallKind, LogCheckCallInput } from "@trenova/graphql/generated/graphql";
import { translate } from "@trenova/shared/i18n/runtime";
import { z } from "zod";

export const CHECK_CALL_LIMITS = {
  city: 100,
  stateCode: 10,
  delayReason: 255,
  note: 1000,
  contactName: 100,
} as const;

const FUTURE_SKEW_SECONDS = 300;

export const checkCallKindSchema = z.enum(["Update", "Arrival", "Departure"]);

export function isStopVisit(kind: CheckCallKind): boolean {
  return kind === "Arrival" || kind === "Departure";
}

function boundedText(limit: number, message: () => string) {
  return z.string().max(limit, { error: message });
}

export const logCheckCallFormSchema = z
  .object({
    shipmentMoveId: z.string().min(1, { error: () => translate("Choose the move") }),
    kind: checkCallKindSchema,
    stopId: z.string().nullable(),
    reportedAt: z.number().int().positive().nullable(),
    city: boundedText(CHECK_CALL_LIMITS.city, () =>
      translate("City cannot be longer than 100 characters"),
    ),
    stateCode: boundedText(CHECK_CALL_LIMITS.stateCode, () =>
      translate("State cannot be longer than 10 characters"),
    ),
    delayed: z.boolean(),
    delayReason: boundedText(CHECK_CALL_LIMITS.delayReason, () =>
      translate("Delay reason cannot be longer than 255 characters"),
    ),
    estimatedArrival: z.number().int().positive().nullable(),
    contactName: boundedText(CHECK_CALL_LIMITS.contactName, () =>
      translate("Contact name cannot be longer than 100 characters"),
    ),
    note: boundedText(CHECK_CALL_LIMITS.note, () =>
      translate("Note cannot be longer than 1000 characters"),
    ),
  })
  .superRefine((values, ctx) => {
    if (isStopVisit(values.kind) && !values.stopId) {
      ctx.addIssue({
        code: "custom",
        path: ["stopId"],
        message: translate("Choose the stop that was reached or left"),
      });
    }
    if (values.delayed && values.delayReason.trim() === "") {
      ctx.addIssue({
        code: "custom",
        path: ["delayReason"],
        message: translate("Say why the load is delayed"),
      });
    }
    if (
      values.reportedAt != null &&
      values.reportedAt > Math.floor(Date.now() / 1000) + FUTURE_SKEW_SECONDS
    ) {
      ctx.addIssue({
        code: "custom",
        path: ["reportedAt"],
        message: translate("The reported time cannot be in the future"),
      });
    }
  });

export type LogCheckCallFormValues = z.infer<typeof logCheckCallFormSchema>;

export function emptyCheckCallForm(shipmentMoveId: string): LogCheckCallFormValues {
  return {
    shipmentMoveId,
    kind: "Update",
    stopId: null,
    reportedAt: null,
    city: "",
    stateCode: "",
    delayed: false,
    delayReason: "",
    estimatedArrival: null,
    contactName: "",
    note: "",
  };
}

function optionalText(value: string): string | undefined {
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

export function toLogCheckCallInput(values: LogCheckCallFormValues): LogCheckCallInput {
  const visit = isStopVisit(values.kind);
  return {
    shipmentMoveId: values.shipmentMoveId,
    kind: values.kind,
    stopId: visit ? (values.stopId ?? undefined) : undefined,
    reportedAt: values.reportedAt ?? undefined,
    city: optionalText(values.city),
    stateCode: optionalText(values.stateCode)?.toUpperCase(),
    delayed: values.delayed,
    delayReason: values.delayed ? optionalText(values.delayReason) : undefined,
    estimatedArrival: values.estimatedArrival ?? undefined,
    contactName: optionalText(values.contactName),
    note: optionalText(values.note),
  };
}

export const issueTrackingLinkFormSchema = z.object({
  recipientEmail: z.union([
    z.literal(""),
    z.email({ error: () => translate("Enter a valid email address") }).max(320),
  ]),
});

export type IssueTrackingLinkFormValues = z.infer<typeof issueTrackingLinkFormSchema>;
