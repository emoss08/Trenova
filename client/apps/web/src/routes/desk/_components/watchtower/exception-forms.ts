import type { WatchtowerResolutionOption } from "@/lib/graphql/watchtower";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { z } from "zod";

/** The changes a person makes to an exception from a dialog. */
export type ExceptionAction = "assign" | "snooze" | "due" | "resolve" | "reopen";

export const MAX_EXCEPTION_NOTE = 2000;
export const MAX_EXCEPTION_COMMENT = 4000;
export const MAX_SNOOZE_SECONDS = 30 * 24 * 60 * 60;
export const MAX_DUE_AHEAD_SECONDS = 90 * 24 * 60 * 60;

export type ExceptionActionValues = {
  assigneeId: string;
  until: number | null;
  dueAt: number | null;
  code: string;
  note: string;
};

export const EMPTY_EXCEPTION_ACTION: ExceptionActionValues = {
  assigneeId: "",
  until: null,
  dueAt: null,
  code: "",
  note: "",
};

/**
 * One schema per action, holding the same limits the server enforces, so a
 * person is told before they submit rather than after. The server stays the
 * authority: these only save a round trip.
 */
export function exceptionActionSchema(
  t: TranslateFn,
  action: ExceptionAction,
  codes: readonly WatchtowerResolutionOption[],
  now: number,
) {
  const note = z
    .string()
    .max(MAX_EXCEPTION_NOTE, { error: t("A note cannot be longer than 2000 characters") });
  const base = z.object({
    assigneeId: z.string(),
    until: z.number().nullable(),
    dueAt: z.number().nullable(),
    code: z.string(),
    note,
  });

  switch (action) {
    case "snooze":
      return base.superRefine((values, ctx) => {
        if (values.until === null) {
          ctx.addIssue({
            code: "custom",
            path: ["until"],
            message: t("Choose when it comes back"),
          });
        } else if (values.until <= now) {
          ctx.addIssue({
            code: "custom",
            path: ["until"],
            message: t("Snooze until a time in the future"),
          });
        } else if (values.until - now > MAX_SNOOZE_SECONDS) {
          ctx.addIssue({
            code: "custom",
            path: ["until"],
            message: t("An exception can be snoozed for at most 30 days"),
          });
        }
      });
    case "due":
      return base.superRefine((values, ctx) => {
        if (values.dueAt !== null && values.dueAt - now > MAX_DUE_AHEAD_SECONDS) {
          ctx.addIssue({
            code: "custom",
            path: ["dueAt"],
            message: t("An exception can be due at most 90 days out"),
          });
        }
      });
    case "resolve":
      return base.superRefine((values, ctx) => {
        const option = codes.find((candidate) => candidate.code === values.code);
        if (option === undefined) {
          ctx.addIssue({ code: "custom", path: ["code"], message: t("Choose a resolution") });
        } else if (option.requiresNote && values.note.trim() === "") {
          ctx.addIssue({ code: "custom", path: ["note"], message: t("Say how it was resolved") });
        }
      });
    default:
      return base;
  }
}
