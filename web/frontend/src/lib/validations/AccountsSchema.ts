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

import { JobFunctionChoiceProps } from "@/lib/choices";
import { StatusChoiceProps } from "@/types";
import { JobTitleFormValues } from "@/types/accounts";
import * as yup from "yup";

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
  emailAddress: yup.string().email().required("Email is required."),
  password: yup.string().required("Password is required."),
});

/**
 * A yup object schema for validating check user email data.
 * @property email - A required string.
 */
export const checkUserEmailSchema = yup.object().shape({
  email: yup.string().email().required("Email is required."),
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
