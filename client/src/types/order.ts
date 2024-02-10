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

import { CodeTypeProps, ShipmentStatusChoiceProps } from "@/lib/choices";
import { StatusChoiceProps } from "@/types/index";
import {
  Control,
  UseFormSetValue,
  UseFormWatch
} from "react-hook-form";
import { BaseModel } from "./organization";

export interface ShipmentControl extends BaseModel {
  id: string;
  autoRateShipment: boolean;
  calculateDistance: boolean;
  enforceRevCode: boolean;
  enforceVoidedComm: boolean;
  generateRoutes: boolean;
  enforceCommodity: boolean;
  autoSequenceStops: boolean;
  autoShipmentTotal: boolean;
  enforceOriginDestination: boolean;
  checkForDuplicateBol: boolean;
  removeShipment: boolean;
}

export type ShipmentControlFormValues = Omit<
  ShipmentControl,
  "id" | "organization" | "created" | "modified"
>;

export interface ShipmentType extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  code: string;
  description?: string | null;
}

export type ShipmentTypeFormValues = Omit<
  ShipmentType,
  "id" | "organization" | "created" | "modified"
>;

export interface ServiceType extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  code: string;
  description?: string | null;
}

export type ServiceTypeFormValues = Omit<
  ServiceType,
  "id" | "organization" | "created" | "modified"
>;

export interface ReasonCode extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  code: string;
  codeType: CodeTypeProps;
  description: string;
}

export type ReasonCodeFormValues = Omit<
  ReasonCode,
  "id" | "organization" | "created" | "modified"
>;

export interface Shipment extends BaseModel {
  id: string;
  proNumber: string;
  shipmentType: string;
  serviceType?: string | null;
  status: ShipmentStatusChoiceProps;
  revenueCode?: string | null;
  originLocation?: string | null;
  originAddress?: string | null;
  originAppointmentWindowStart: string;
  originAppointmentWindowEnd: string;
  destinationLocation?: string | null;
  destinationAddress?: string | null;
  destinationAppointmentWindowStart: string;
  destinationAppointmentWindowEnd: string;
  ratingUnits: number;
  rate?: string | null;
  mileage?: number | null;
  otherChargeAmount?: string | null;
  freightChargeAmount?: string | null;
  rateMethod?: string | null;
  customer: string;
  pieces?: number | null;
  weight?: string | null;
  readyToBill: boolean;
  billDate?: string | null;
  shipDate?: string | null;
  billed: boolean;
  transferredToBilling: boolean;
  billingTransferDate?: string | null;
  subTotal: string;
  trailer?: string | null;
  trailerType: string;
  tractorType?: string | null;
  commodity?: string | null;
  enteredBy: string;
  hazardousMaterial?: string | null;
  temperatureMin?: string | null;
  temperatureMax?: string | null;
  bolNumber: string;
  consigneeRefNumber?: string | null;
  comment?: string | null;
  voidedComm?: string | null;
  autoRate: boolean;
  currentSuffix?: string | null;
  formulaTemplate?: string | null;
  entryMethod: string;
  copyAmount?: number | null;
}

export type ShipmentFormValues = Omit<
  Shipment,
  | "id"
  | "organization"
  | "billDate"
  | "shipDate"
  | "billed"
  | "transferredToBilling"
  | "billingTransferDate"
  | "currentSuffix"
  | "created"
  | "modified"
>;

export type ShipmentFormProps = {
  control: Control<ShipmentFormValues>;
  setValue: UseFormSetValue<ShipmentFormValues>;
  watch: UseFormWatch<ShipmentFormValues>;
};

export type ShipmentSearchForm = {
  searchQuery: string;
  statusFilter: string;
};

export interface FormulaTemplate extends BaseModel {
  id: string;
  name: string;
  formulaText: string;
  description?: string;
  templateType: string;
  customer?: string | null;
  shipmentType?: string | null;
  autoApply: boolean;
}
