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
import { LocationCategoryFormValues } from "@/types/location";
import * as yup from "yup";

export const locationCategorySchema: yup.ObjectSchema<LocationCategoryFormValues> =
  yup.object().shape({
    name: yup
      .string()
      .required("Name is required")
      .max(100, "Name cannot be more than 100 characters"),
    description: yup
      .string()
      .max(500, "Description cannot be more than 500 characters")
      .notRequired(),
  });
