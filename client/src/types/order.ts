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

import { CodeTypeProps } from "@/lib/choices";
import { StatusChoiceProps } from "@/types/index";
import { Control, UseFormSetValue, UseFormWatch } from "react-hook-form";
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
  status: string;
  revenueCode?: string | null;
  originLocation?: string | null;
  originAddress?: string;
  originAppointmentWindowStart: Date;
  originAppointmentWindowEnd: Date;
  destinationLocation?: string | null;
  destinationAddress?: string;
  destinationAppointmentWindowStart: Date;
  destinationAppointmentWindowEnd: Date;
  ratingUnits: number;
  rate?: string | null;
  mileage?: number | null;
  otherChargeAmount: string;
  freightChargeAmount?: string | null;
  rateMethod?: string;
  customer: string;
  pieces: number;
  weight: string;
  readyToBill: boolean;
  billDate?: Date | null;
  shipDate?: Date | null;
  billed: boolean;
  transferredToBilling: boolean;
  billingTransferDate?: Date | null;
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
  consigneeRefNumber?: string;
  comment?: string;
  voidedComm?: string;
  autoRate: boolean;
  currentSuffix?: string;
  formulaTemplate?: string | null;
  entryMethod: string;
}

export type ShipmentFormValues = Omit<
  Shipment,
  "id" | "organization" | "created" | "modified"
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
