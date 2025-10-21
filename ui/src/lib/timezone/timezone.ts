/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { ChoiceProps } from "@/types/common";
import tz from "./timezone.json";

export const TIMEZONES: ChoiceProps<string>[] = tz;

// Create a type for each value in the TIMEZONES array
export type TimezoneChoices = (typeof TIMEZONES)[number]["value"];
