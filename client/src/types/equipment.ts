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
import { IChoiceProps, StatusChoiceProps } from "@/types/index";
import { FleetCode } from "./dispatch";
import { BaseModel } from "./organization";
import { Worker } from "./worker";

export interface EquipmentType extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  code: string;
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
  color?: string;
}

export type EquipmentTypeFormValues = Omit<
  EquipmentType,
  | "organizationId"
  | "businessUnit"
  | "createdAt"
  | "updatedAt"
  | "id"
  | "version"
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

export type EquipmentStatus =
  | "Available"
  | "OutOfService"
  | "AtMaintenance"
  | "Sold"
  | "Lost";

export const equipmentStatusChoices = [
  {
    value: "Available",
    label: "Available",
    color: "#16a34a",
  },
  {
    value: "OutOfService",
    label: "Out of Service",
    color: "#dc2626",
  },
  {
    value: "AtMaintenance",
    label: "At Maintenance",
    color: "#9333ea",
  },
  {
    value: "Sold",
    label: "Sold",
    color: "#2563eb",
  },
  {
    value: "Lost",
    label: "Lost",
    color: "#ca8a04",
  },
] satisfies IChoiceProps<EquipmentStatus>[];

export interface Trailer extends BaseModel {
  id: string;
  code: string;
  status: EquipmentStatus;
  equipmentTypeId: string;
  equipmentManufacturerId?: string | null;
  model?: string;
  year?: number | null;
  vin?: string;
  fleetCodeId: string;
  stateId?: string | null;
  licensePlateNumber?: string;
  lastInspectionDate?: string;
  registrationNumber?: string;
  registrationStateId?: string | null;
  registrationExpirationDate?: string | null;
  edges?: {
    equipmentType?: EquipmentType;
    equipmentManufacturer?: EquipmentManufacturer;
    primaryWorker?: Worker;
    secondaryWorker?: Worker;
    fleetCode?: FleetCode;
  };
}

export type TrailerFormValues = Omit<
  Trailer,
  | "id"
  | "organizationId"
  | "businessUnitId"
  | "version"
  | "edges"
  | "equipTypeName"
  | "createdAt"
  | "updatedAt"
>;

export interface Tractor extends BaseModel {
  id: string;
  code: string;
  equipmentTypeId: string;
  status: EquipmentStatus;
  licensePlateNumber?: string;
  vin?: string;
  equipmentManufacturerId?: string | null;
  model?: string;
  year?: number | null;
  state?: string;
  leased: boolean;
  leasedDate?: string | null;
  primaryWorkerId: string;
  secondaryWorkerId?: string | null;
  fleetCodeId: string;
  edges?: {
    equipmentType?: EquipmentType;
    equipmentManufacturer?: EquipmentManufacturer;
    primaryWorker?: Worker;
    secondaryWorker?: Worker;
    fleetCode?: FleetCode;
  };
}

export type TractorFormValues = Omit<
  Tractor,
  | "id"
  | "organizationId"
  | "businessUnitId"
  | "createdAt"
  | "updatedAt"
  | "edges"
  | "version"
>;

export type EquipmentClass = "TRACTOR" | "TRAILER";
