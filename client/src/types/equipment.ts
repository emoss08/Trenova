/*
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

import { EquipmentClassChoiceProps } from "@/lib/choices";
import { StatusChoiceProps } from "@/types/index";
import { BaseModel } from "./organization";

export interface EquipmentType extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  name: string;
  description?: string;
  costPerMile?: number;
  equipmentClass: EquipmentClassChoiceProps;
  fixedCost?: number;
  variableCost?: number;
  height?: number;
  length?: number;
  width?: number;
  weight?: number;
  idlingFuelUsage?: number;
  exemptFromTolls: boolean;
}

export type EquipmentTypeFormValues = Omit<
  EquipmentType,
  "organizationId" | "businessUnit" | "createdAt" | "updatedAt" | "id"
>;

export interface EquipmentManufacturer extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  name: string;
  description?: string;
}

export type EquipmentManufacturerFormValues = Pick<
  EquipmentManufacturer,
  "name" | "description" | "status"
>;

export type EquipmentStatus = "A" | "OOS" | "AM" | "S" | "L";

export const equipmentStatusChoices = [
  {
    value: "A",
    label: "Available",
    color: "#16a34a",
  },
  {
    value: "OOS",
    label: "Out of Service",
    color: "#dc2626",
  },
  {
    value: "AM",
    label: "At Maintenance",
    color: "#9333ea",
  },
  {
    value: "S",
    label: "Sold",
    color: "#2563eb",
  },
  {
    value: "L",
    label: "Lost",
    color: "#ca8a04",
  },
];

export interface Trailer extends BaseModel {
  id: string;
  code: string;
  status: EquipmentStatus;
  equipmentType: string;
  manufacturer?: string;
  make?: string;
  model?: string;
  year?: number;
  vinNumber?: string;
  fleetCode?: string;
  state?: string;
  licensePlateNumber?: string;
  licensePlateState?: string;
  lastInspection?: string;
  registrationNumber?: string;
  registrationState?: string;
  registrationExpiration?: string;
  isLeased: boolean;
  timesUsed: number;
  equipTypeName: string;
}

export type TrailerFormValues = Omit<
  Trailer,
  | "id"
  | "timesUsed"
  | "equipTypeName"
  | "organization"
  | "businessUnit"
  | "created"
  | "modified"
>;

export interface Tractor extends BaseModel {
  id: string;
  code: string;
  equipmentType: string;
  status: string;
  licensePlateNumber?: string;
  vinNumber?: string;
  manufacturer?: string;
  model?: string;
  year?: number;
  state?: string;
  leased: boolean;
  leasedDate?: string;
  primaryWorker?: string;
  secondaryWorker?: string;
  hosExempt: boolean;
  ownerOperated: boolean;
  fleetCode?: string;
}

export type TractorFormValues = Omit<
  Tractor,
  "id" | "organization" | "businessUnit" | "created" | "modified"
>;

export type EquipmentClass = "TRACTOR" | "TRAILER";
