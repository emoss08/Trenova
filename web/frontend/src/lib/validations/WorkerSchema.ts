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

import { type StatusChoiceProps } from "@/types";
import {
  EmploymentHistoryFormValues,
  EnumEmploymentVerificationStatus,
  EnumWorkerType,
  WorkerFormValues,
  type WorkerProfileFormValues,
} from "@/types/worker";
import { array, mixed, object, string, type ObjectSchema } from "yup";

const workerProfileSchema: ObjectSchema<WorkerProfileFormValues> =
  object().shape({
    race: string(),
    sex: string(),
    dateOfBirth: string().required("Date of Birth is required"),
    stateId: string().required("License State is required"),
    licenseExpirationDate: string().required(
      "License Expiration Date is required",
    ),
    endorsements: string().required("Endorsements is required"),
    hazmatExpirationDate: string().nullable(),
    hireDate: string().nullable(),
    terminationDate: string().nullable(),
    physicalDueDate: string().nullable(),
    mvrDueDate: string().nullable(),
  });

const employmentHistorySchema: ObjectSchema<EmploymentHistoryFormValues> =
  object().shape({
    employerName: string().required("Employer Name is required"),
    employerAddress: string().required("Employer Address is required"),
    employerContactInfo: string().required("Employer Contact Info is required"),
    startDate: string().required("Start Date is required"),
    endDate: string().required("End Date is required"),
    reasonForLeaving: string(),
    verificationStatus: mixed<EnumEmploymentVerificationStatus>()
      .required("Verification Status is required")
      .oneOf(Object.values(EnumEmploymentVerificationStatus)),
  });

export const workerSchema: ObjectSchema<WorkerFormValues> = object().shape({
  status: string<StatusChoiceProps>().required("Status is required"),
  code: string().required("Code is required"),
  workerType: mixed<EnumWorkerType>()
    .required("Worker Type is required")
    .oneOf(Object.values(EnumWorkerType)),
  firstName: string().required("First Name is required"),
  lastName: string().required("Last Name is required"),
  addressLine1: string(),
  addressLine2: string(),
  city: string(),
  stateId: string().nullable(),
  zipCode: string(),
  managerId: string().nullable(),
  profilePictureUrl: string(),
  workerProfile: workerProfileSchema,
  employmentHistory: array().of(employmentHistorySchema),
});
