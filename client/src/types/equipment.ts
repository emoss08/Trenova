import type { EquipmentClassChoiceProps } from "@/lib/choices";
import type { IChoiceProps, StatusChoiceProps } from "@/types/index";
import { type FleetCode } from "./dispatch";
import { type BaseModel } from "./organization";
import { type Worker } from "./worker";

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
  lastInspectionDate?: string | null;
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
