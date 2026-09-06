import { z } from "zod";

export const availabilityPreferenceSchema = z.enum(["Preferred", "Available", "Unavailable"]);
export type AvailabilityPreferenceValue = z.infer<typeof availabilityPreferenceSchema>;

export const shiftSwapStatusSchema = z.enum([
  "Proposed",
  "Accepted",
  "Declined",
  "Approved",
  "Rejected",
  "Withdrawn",
]);
export type ShiftSwapStatusValue = z.infer<typeof shiftSwapStatusSchema>;

const dayMaskSchema = z
  .string()
  .regex(/^[01]{7}$/, "Days must be seven characters of 0 or 1, starting on Sunday");

/**
 * A repeating working pattern. The mask is stored rather than an array of days
 * so a pattern reads at a glance and compares without unpacking, and a pattern
 * with no working days is refused: it is a way to roster somebody onto nothing
 * and never notice.
 */
export const shiftTemplateFormSchema = z
  .object({
    code: z.string().min(1, "A code is required").max(20),
    name: z.string().min(1, "A name is required").max(100),
    description: z.string().max(500).nullable(),
    color: z.string().max(10).nullable(),
    daysOfWeek: dayMaskSchema,
    // A clock time and a length in hours: that is how a shift is described
    // out loud, and the minutes the server stores are derived from them.
    startTime: z.string().regex(/^\d{2}:\d{2}$/, "A start time is required"),
    durationHours: z
      .number()
      .min(0.25, "A shift is at least a quarter of an hour")
      .max(24, "A shift is at most a day"),
    cycleWeeks: z.number().int().min(1, "A rotation is 1 to 8 weeks").max(8),
    status: z.enum(["Active", "Inactive"]),
  })
  .refine((values) => values.daysOfWeek.includes("1"), {
    message: "A shift needs at least one working day",
    path: ["daysOfWeek"],
  });
export type ShiftTemplateFormValues = z.infer<typeof shiftTemplateFormSchema>;

/**
 * Putting a worker on a pattern. The offset is what makes an A/B rotation one
 * template and two assignments rather than two near-identical templates.
 */
export const assignShiftFormSchema = z.object({
  workerId: z.string().min(1, "Choose a worker"),
  shiftTemplateId: z.string().min(1, "Choose a shift"),
  effectiveFrom: z.number().int().positive("An effective date is required"),
  cycleOffsetWeeks: z.number().int().min(0).max(7),
  notes: z.string().max(500).nullable(),
});
export type AssignShiftFormValues = z.infer<typeof assignShiftFormSchema>;

/**
 * One driver asking another to take a day. The counterparty is optional: a
 * hand-off to whoever the office finds is still a request the office has to
 * see, and refusing it would push the driver to phone somebody instead.
 */
export const proposeShiftSwapFormSchema = z
  .object({
    requestingWorkerId: z.string().min(1, "Choose a worker"),
    counterpartyWorkerId: z.string().nullable(),
    shiftDate: z.number().int().positive("Choose the day being given up"),
    counterpartyShiftDate: z.number().int().positive().nullable(),
    reason: z.string().max(255).nullable(),
  })
  // Swapping with yourself changes nothing and shows on the board as cover
  // somebody arranged.
  .refine(
    (values) =>
      !values.counterpartyWorkerId || values.counterpartyWorkerId !== values.requestingWorkerId,
    {
      message: "A swap has to be with somebody else",
      path: ["counterpartyWorkerId"],
    },
  );
export type ProposeShiftSwapFormValues = z.infer<typeof proposeShiftSwapFormSchema>;
