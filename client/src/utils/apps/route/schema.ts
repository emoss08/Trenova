/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import * as Yup from "yup";
import { ObjectSchema } from "yup";
import { RouteControlFormValues } from "@/types/apps/route";
import {
  DistanceMethodChoiceProps,
  RouteModelChoiceProps,
} from "@/utils/apps/route/index";

export const routeControlSchema: ObjectSchema<RouteControlFormValues> =
  Yup.object().shape({
    distance_method: Yup.string<DistanceMethodChoiceProps>().required(
      "Distance method is required"
    ),
    mileage_unit: Yup.string<RouteModelChoiceProps>().required(
      "Mileage unit is required"
    ),
    generate_routes: Yup.boolean().required("Generate routes is required"),
  });
