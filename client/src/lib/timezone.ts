import { TChoiceProps } from "@/types";
import tz from "./timezone.json";

export const TIMEZONES: TChoiceProps[] = tz;

// Create a type for each value in the TIMEZONES array
export type TimezoneChoices = (typeof TIMEZONES)[number]["value"];
