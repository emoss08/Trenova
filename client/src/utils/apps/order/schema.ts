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
import { OrderControlFormValues } from "@/types/apps/order";

export const orderControlSchema: ObjectSchema<OrderControlFormValues> =
  Yup.object().shape({
    auto_rate_orders: Yup.boolean().required("Auto Rate Orders is required"),
    calculate_distance: Yup.boolean().required(
      "Calculate Distance is required"
    ),
    enforce_rev_code: Yup.boolean().required("Enforce Rev Code is required"),
    enforce_voided_comm: Yup.boolean().required(
      "Enforce Voided Comm is required"
    ),
    generate_routes: Yup.boolean().required("Generate Routes is required"),
    enforce_commodity: Yup.boolean().required("Enforce Commodity is required"),
    auto_sequence_stops: Yup.boolean().required(
      "Auto Sequence Stops is required"
    ),
    auto_order_total: Yup.boolean().required("Auto Order Total is required"),
    enforce_origin_destination: Yup.boolean().required(
      "Enforce Origin Destination is required"
    ),
    check_for_duplicate_bol: Yup.boolean().required(
      "Check for Duplicate BOL is required"
    ),
    remove_orders: Yup.boolean().required("Remove Orders is required"),
  });
