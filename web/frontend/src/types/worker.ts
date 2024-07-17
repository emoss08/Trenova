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

import { type BaseModel } from "@/types/organization";
import { IChoiceProps, type StatusChoiceProps } from ".";

interface WorkerProfile extends BaseModel {
  race?: string;
  sex?: string;
  dateOfBirth: string;
  stateId: string;
  licenseExpirationDate: string;
  endorsements?: string;
  hazmatExpirationDate?: string | null;
  hireDate?: string | null;
  terminationDate?: string | null;
  physicalDueDate?: string | null;
  mvrDueDate?: string | null;
}

export type WorkerProfileFormValues = Omit<
  WorkerProfile,
  "organizationId" | "createdAt" | "updatedAt" | "id" | "version"
>;

export enum EnumWorkerType {
  Driver = "Employee",
  Warehouse = "Contractor",
}

/* type for worker type */
type WorkerType = "Employee" | "Contractor";

export const workerTypeChoices = [
  { value: "Employee", label: "Employee", color: "#2563eb" },
  { value: "Contractor", label: "Contractor", color: "#15803d" },
] satisfies ReadonlyArray<IChoiceProps<WorkerType>>;

export enum EnumEmploymentVerificationStatus {
  Verified = "Verified",
  Pending = "Pending",
  Failed = "Failed",
}

/* type for employment verification status */
export type EmploymentVerificationStatus = "Verified" | "Pending" | "Failed";

export const employmentVerificationChoices = [
  { value: "Verified", label: "Verified", color: "#2563eb" },
  { value: "Pending", label: "API", color: "#15803d" },
  { value: "Failed", label: "Local", color: "#b91c1c" },
] satisfies ReadonlyArray<IChoiceProps<EmploymentVerificationStatus>>;

interface EmploymentHistory extends BaseModel {
  employerName: string;
  employerAddress: string;
  employerContactInfo: string;
  startDate: string;
  endDate: string;
  reasonForLeaving?: string;
  verificationStatus: EnumEmploymentVerificationStatus;
}

export type EmploymentHistoryFormValues = Omit<
  EmploymentHistory,
  "organizationId" | "createdAt" | "updatedAt" | "id" | "version"
>;

export interface Worker extends BaseModel {
  id: string;
  code: string;
  status: StatusChoiceProps;
  workerType: EnumWorkerType;
  firstName: string;
  lastName: string;
  addressLine1?: string;
  addressLine2?: string;
  city?: string;
  stateId?: string | null;
  zipCode?: string;
  managerId?: string | null;
  profilePictureUrl?: string;
  fleetCodeId?: string | null;
  workerProfile: WorkerProfile;
  employmentHistory?: EmploymentHistory[];
}

export type WorkerFormValues = Omit<
  Worker,
  | "organizationId"
  | "createdAt"
  | "updatedAt"
  | "id"
  | "version"
  | "workerProfile"
  | "employmentHistory"
> & {
  workerProfile: WorkerProfileFormValues;
  employmentHistory?: EmploymentHistoryFormValues[];
};
