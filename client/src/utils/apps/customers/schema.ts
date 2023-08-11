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

import { ObjectSchema } from "yup";
import * as Yup from "yup";
import { StatusChoiceProps, YesNoChoiceProps } from "@/types";
import { CustomerFormValues } from "@/types/apps/customer";

export const customerSchema: ObjectSchema<CustomerFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    code: Yup.string().required("Code is required"),
    name: Yup.string().required("Name is required"),
    address_line_1: Yup.string().notRequired(),
    address_line_2: Yup.string().notRequired(),
    city: Yup.string().notRequired(),
    state: Yup.string().notRequired(),
    zip_code: Yup.string().notRequired(),
    has_customer_portal: Yup.string<YesNoChoiceProps>().required(
      "Has Customer Portal is required",
    ),
    auto_mark_ready_to_bill: Yup.string<YesNoChoiceProps>().required(
      "Auto Mark Ready to Bill is required",
    ),
  });
