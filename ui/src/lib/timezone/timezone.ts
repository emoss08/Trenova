import { ChoiceProps } from "@/types/common";
import tz from "./timezone.json";

export const TIMEZONES: ChoiceProps<string>[] = tz;

// Create a type for each value in the TIMEZONES array
export type TimezoneChoices = (typeof TIMEZONES)[number]["value"];
