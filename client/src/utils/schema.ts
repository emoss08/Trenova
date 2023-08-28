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

/**
 * A Yup object schema for validating login data.
 * @property username - A required string.
 * @property password - A required string.
 */
export const LoginSchema = Yup.object().shape({
  username: Yup.string().required("Username is required."),
  password: Yup.string().required("Password is required."),
});

/**
 * A Yup object schema for validating user profile data.
 * @property profile - An object that includes properties:
 *                     first_name, last_name, address_line_1, city, state, zip_code, phone_number.
 */
export const UserSchema = Yup.object().shape({
  profile: Yup.object().shape({
    firstName: Yup.string().required("First name is required"),
    lastName: Yup.string().required("Last name is required"),
    addressLine1: Yup.string().required("Address Line 1 is required"),
    city: Yup.string().required("City is required"),
    state: Yup.string().required("State is required"),
    zipCode: Yup.string().required("Zip Code is required"),
    phoneNumber: Yup.string()
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

/**
 * A Yup object schema for validating data related to exporting a model.
 * @property file_format - A required string.
 * @property columns - A required array of strings.
 */
export const ExportModelSchema = Yup.object().shape({
  fileFormat: Yup.string().required("File format is required"),
  columns: Yup.array().of(Yup.string()).required("Columns are required"),
});
