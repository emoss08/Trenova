/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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
