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

import { StatusChoiceProps } from "@/types";
import {
  LocationCategoryFormValues,
  LocationCommentFormValues,
  LocationContactFormValues,
  LocationFormValues,
} from "@/types/location";
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
    color: yup
      .string()
      .max(100, "Color cannot be more than 100 characters")
      .notRequired(),
  });

const locationCommentSchema: yup.ObjectSchema<LocationCommentFormValues> = yup
  .object()
  .shape({
    commentType: yup.string().required("Comment Type is required"),
    comment: yup.string().required("Comment is required"),
  });

const locationContactSchema: yup.ObjectSchema<LocationContactFormValues> = yup
  .object()
  .shape({
    name: yup.string().required("Name is required"),
    email: yup.string().email().notRequired(),
    phone: yup
      .string()
      .notRequired()
      .test(
        "phone_number_format",
        "Phone number must be in the format (xxx) xxx-xxxx",
        (value) => {
          if (!value) {
            return true;
          } // if the string is null or undefined, skip the test
          const regex = /^\(?([0-9]{3})\)?[-. ]?([0-9]{3})[-. ]?([0-9]{4})$/;
          return regex.test(value); // apply the regex test if string exists
        },
      ),
    fax: yup
      .string()
      .notRequired()
      .test(
        "phone_number_format",
        "Phone number must be in the format (xxx) xxx-xxxx",
        (value) => {
          if (!value) {
            return true;
          } // if the string is null or undefined, skip the test
          const regex = /^\(?([0-9]{3})\)?[-. ]?([0-9]{3})[-. ]?([0-9]{4})$/;
          return regex.test(value); // apply the regex test if string exists
        },
      ),
  });

export const locationSchema: yup.ObjectSchema<LocationFormValues> = yup
  .object()
  .shape({
    status: yup.string<StatusChoiceProps>().required("Status is required"),
    code: yup
      .string()
      .max(10, "Code cannot be more than 10 characters")
      .required("Code is required"),
    locationCategory: yup.string().optional(),
    name: yup
      .string()
      .required("Name is required")
      .max(100, "Name cannot be more than 255 characters"),
    depot: yup.string().optional(),
    description: yup.string().notRequired(),
    addressLine1: yup.string().required("Address Line 1 is required"),
    addressLine2: yup.string().notRequired(),
    city: yup.string().required("City is required"),
    state: yup.string().required("State is required"),
    zipCode: yup.string().required("Zip Code is required"),
    locationComments: yup.array().of(locationCommentSchema).notRequired(),
    locationContacts: yup.array().of(locationContactSchema).notRequired(),
  });
