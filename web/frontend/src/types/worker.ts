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
