/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import {
  DistanceMethodChoiceProps,
  RouteDistanceUnitProps,
} from "@/lib/choices";
import { RouteControlFormValues } from "@/types/route";
import * as Yup from "yup";
import { ObjectSchema } from "yup";

export const routeControlSchema: ObjectSchema<RouteControlFormValues> =
  Yup.object().shape({
    distanceMethod: Yup.string<DistanceMethodChoiceProps>().required(
      "Distance method is required",
    ),
    mileageUnit: Yup.string<RouteDistanceUnitProps>().required(
      "Mileage unit is required",
    ),
    generateRoutes: Yup.boolean().required("Generate routes is required"),
  });
