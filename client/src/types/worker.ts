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

import { BaseModel } from "@/types/organization";

export interface Worker extends BaseModel {
  id: string;
  code: string;
  isActive: boolean;
  workerType: string;
  firstName: string;
  lastName: string;
  addressLine1: string;
  addressLine2?: string;
  city: string;
  state: string;
  fleetCode: string;
  zipCode: string;
  depot?: string | null;
  manager: string;
  enteredBy: string;
  profile: WorkerProfile;
  contacts: WorkerContact[];
  comments: WorkerComment[];
}

export interface WorkerProfile extends BaseModel {
  worker: string;
  race?: string;
  sex?: string;
  dateOfBirth?: Date | null;
  licenseState?: string;
  licenseExpirationDate?: Date | null;
  endorsements?: string;
  hazmatExpirationDate?: Date | null;
  hm126ExpirationDate?: Date | null;
  hireDate?: Date | null;
  terminationDate?: Date | null;
  reviewDate?: Date | null;
  physicalDueDate?: Date | null;
  mvrDueDate?: Date | null;
  medicalCertDate?: Date | null;
}

export interface WorkerContact extends BaseModel {
  id: string;
  worker: string;
  name: string;
  phone?: number | null;
  email?: string;
  relationship?: string;
  isPrimary: boolean;
  mobilePhone?: number | null;
}

export interface WorkerComment extends BaseModel {
  id: string;
  worker: string;
  commentType: string;
  comment: string;
  enteredBy: string;
}

export interface WorkerTimeAway extends BaseModel {
  id: string;
  worker: string;
  startDate: Date;
  endDate: Date;
  leaveType: string;
}

export interface WorkerHOS extends BaseModel {
  id: string;
  worker: string;
  driveTime: number;
  offDutyTime: number;
  sleeperBerthTime: number;
  onDutyTime: number;
  violationTime: number;
  currentStatus: string;
  currentLocation: string;
  seventyHourTime: number;
  milesDriven: number;
  logDate: Date;
  lastResetDate: Date;
}
