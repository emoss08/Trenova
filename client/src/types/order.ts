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

export type Shipment = {
  mileage: number;
  comment: string;
  proNumber: string;
  originAppointmentWindowStart: string;
  billed: boolean;
  temperatureMin: null | number;
  id: string;
  transferredToBilling: boolean;
  destinationLocation: string;
  subTotalCurrency: string;
  destinationAddress: string;
  movements: string[];
  equipmentType: string;
  originAppointmentWindowEnd: string;
  readyToBill: boolean;
  orderComments: string[];
  freightChargeAmount: string;
  rateMethod: string;
  commodity: null | string;
  subTotal: string;
  bolNumber: string;
  additionalCharges: any[];
  enteredBy: string;
  billingTransferDate: null | string;
  weight: string;
  temperatureMax: null | number;
  voidedComm: string;
  originAddress: string;
  freightChargeAmountCurrency: string;
  otherChargeAmount: string;
  orderDocumentation: any[];
  hazmat: null | string;
  status: string;
  otherChargeAmountCurrency: string;
  destinationAppointmentWindowStart: string;
  destinationAppointmentWindowEnd: string;
  customer: string;
  pieces: number;
  autoRate: boolean;
  orderType: string;
  consigneeRefNumber: string;
  billDate: null | string;
  rate: null | string;
  revenueCode: null | string;
  originLocation: string;
};

export type OrderType = {
  organization: string;
  businessUnit: string;
  id: string;
  isActive: boolean;
  name: string;
  description: string;
  created: string;
  modified: string;
};
