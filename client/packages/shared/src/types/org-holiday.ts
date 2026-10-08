import { z } from "zod";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { translate } from "@trenova/shared/i18n/runtime";

export const orgHolidayKindSchema = z.enum(["Holiday", "Blackout"]);
export type OrgHolidayKind = z.infer<typeof orgHolidayKindSchema>;

export const ORG_HOLIDAY_KIND_LABELS: Record<OrgHolidayKind, string> = defineLabels({
  Holiday: "Holiday",
  Blackout: "Blackout",
});

/** What each kind does to a PTO request, shown beside the kind picker. */
export const ORG_HOLIDAY_KIND_HINTS: Record<OrgHolidayKind, string> = defineLabels({
  Holiday: "Not counted against a request when the policy skips weekends.",
  Blackout: "Time off cannot be requested on this date.",
});

export const orgHolidayFormSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, { error: () => translate("Name is required") })
    .max(100, { error: () => translate("Name cannot exceed 100 characters") }),
  holidayDate: z
    .number()
    .int()
    .positive({ error: () => translate("Choose the date") }),
  kind: orgHolidayKindSchema,
  recursAnnually: z.boolean(),
  description: z
    .string()
    .nullable()
    .transform((value) => {
      const trimmed = value?.trim() ?? "";
      return trimmed === "" ? null : trimmed;
    })
    .pipe(
      z
        .string()
        .max(500, { error: () => translate("Description cannot exceed 500 characters") })
        .nullable(),
    ),
});

export type OrgHolidayFormValues = z.infer<typeof orgHolidayFormSchema>;
