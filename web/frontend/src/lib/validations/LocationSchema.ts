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

import { StatusChoiceProps } from "@/types";
import {
  LocationCategoryFormValues,
  LocationCommentFormValues,
  LocationContactFormValues,
  LocationFormValues,
} from "@/types/location";
import * as yup from "yup";

export const LocationCategorySchema: yup.ObjectSchema<LocationCategoryFormValues> =
  yup.object().shape({
    name: yup
      .string()
      .required("Name is required")
      .max(100, "Name cannot be more than 100 characters"),
    description: yup
      .string()
      .max(500, "Description cannot be more than 500 characters"),
    color: yup.string().max(100, "Color cannot be more than 100 characters"),
  });

const LocationCommentSchema: yup.ObjectSchema<LocationCommentFormValues> = yup
  .object()
  .shape({
    commentTypeId: yup.string().required("Comment Type is required"),
    comment: yup.string().required("Comment is required"),
    userId: yup.string().required("User is required"),
  });

const LocationContactSchema: yup.ObjectSchema<LocationContactFormValues> = yup
  .object()
  .shape({
    name: yup.string().required("Name is required"),
    emailAddress: yup.string().email(),
    phoneNumber: yup
      .string()
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

export const LocationSchema: yup.ObjectSchema<LocationFormValues> = yup
  .object()
  .shape({
    status: yup.string<StatusChoiceProps>().required("Status is required"),
    code: yup.string().optional(),
    locationCategoryId: yup.string().nullable(),
    name: yup
      .string()
      .required("Name is required")
      .max(100, "Name cannot be more than 255 characters"),
    description: yup.string(),
    addressLine1: yup.string().required("Address Line 1 is required"),
    addressLine2: yup.string(),
    city: yup.string().required("City is required"),
    stateId: yup.string().required("State is required"),
    postalCode: yup.string().required("Zip Code is required"),
    comments: yup.array().of(LocationCommentSchema).notRequired(),
    contacts: yup.array().of(LocationContactSchema).notRequired(),
  });
