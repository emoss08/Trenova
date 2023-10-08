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

import * as yup from "yup";
import { JobTitleFormValues } from "@/types/accounts";
import { StatusChoiceProps } from "@/types";
import { JobFunctionChoiceProps } from "@/lib/choices";

export const jobTitleSchema: yup.ObjectSchema<JobTitleFormValues> = yup
  .object()
  .shape({
    status: yup.string<StatusChoiceProps>().required("Status is required"),
    name: yup.string().required("Name is required"),
    description: yup.string().notRequired(),
    jobFunction: yup
      .string<JobFunctionChoiceProps>()
      .required("Job Function is required"),
  });

/**
 * A yup object schema for validating login data.
 * @property username - A required string.
 * @property password - A required string.
 */
export const userAuthSchema = yup.object().shape({
  username: yup.string().required("Username is required."),
  password: yup.string().required("Password is required."),
});

export const resetPasswordSchema: yup.ObjectSchema<{
  email: string;
}> = yup.object().shape({
  email: yup.string().email().required("Email is required."),
});

/**
 * A yup object schema for validating user profile data.
 * @property profile - An object that includes properties:
 *                     first_name, last_name, address_line_1, city, state, zip_code, phone_number.
 */
export const UserSchema = yup.object().shape({
  profile: yup.object().shape({
    firstName: yup.string().required("First name is required"),
    lastName: yup.string().required("Last name is required"),
    addressLine1: yup.string().required("Address Line 1 is required"),
    city: yup.string().required("City is required"),
    state: yup.string().required("State is required"),
    zipCode: yup.string().required("Zip Code is required"),
    phoneNumber: yup
      .string()
      .nullable()
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
  }),
});
