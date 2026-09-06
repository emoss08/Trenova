import { z } from "zod";

export const orgHolidayKindSchema = z.enum(["Holiday", "Blackout"]);
export type OrgHolidayKind = z.infer<typeof orgHolidayKindSchema>;

export const ORG_HOLIDAY_KIND_LABELS: Record<OrgHolidayKind, string> = {
  Holiday: "Holiday",
  Blackout: "Blackout",
};

/** What each kind does to a PTO request, shown beside the kind picker. */
export const ORG_HOLIDAY_KIND_HINTS: Record<OrgHolidayKind, string> = {
  Holiday: "Not counted against a request when the policy skips weekends.",
  Blackout: "Time off cannot be requested on this date.",
};

export const orgHolidayFormSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, { message: "Name is required" })
    .max(100, { message: "Name cannot exceed 100 characters" }),
  holidayDate: z.number().int().positive({ message: "Choose the date" }),
  kind: orgHolidayKindSchema,
  recursAnnually: z.boolean(),
  description: z
    .string()
    .nullable()
    .transform((value) => {
      const trimmed = value?.trim() ?? "";
      return trimmed === "" ? null : trimmed;
    })
    .pipe(z.string().max(500, { message: "Description cannot exceed 500 characters" }).nullable()),
});

export type OrgHolidayFormValues = z.infer<typeof orgHolidayFormSchema>;
