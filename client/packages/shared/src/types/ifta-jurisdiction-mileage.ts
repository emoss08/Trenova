import { translate } from "@trenova/shared/i18n/runtime";
import { z } from "zod";
import { positiveDecimalString } from "./decimal";

export const IFTA_MILES_SCALE = 2;
export const IFTA_MILEAGE_NOTES_MAX = 500;

export function createIftaMileageEntryFormSchema(today: number) {
  return z.object({
    tractorId: z.string().min(1, { message: translate("Choose the tractor that ran the miles") }),
    jurisdictionId: z
      .string()
      .min(1, { message: translate("Choose the jurisdiction the miles were run in") }),
    traveledAt: z
      .number()
      .int()
      .positive({ message: translate("Enter the day the miles were run") })
      .max(today, { message: translate("Miles cannot be run in the future") }),
    miles: positiveDecimalString(
      IFTA_MILES_SCALE,
      "Enter miles above zero with up to two decimals",
    ),
    loaded: z.boolean(),
    notes: z
      .string()
      .max(IFTA_MILEAGE_NOTES_MAX, {
        message: `Keep notes under ${IFTA_MILEAGE_NOTES_MAX} characters`,
      })
      .nullable(),
  });
}

export type IftaMileageEntryFormValues = z.infer<
  ReturnType<typeof createIftaMileageEntryFormSchema>
>;
