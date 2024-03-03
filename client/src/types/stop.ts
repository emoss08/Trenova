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

import { ShipmentStatusChoiceProps } from "@/lib/choices";
import { StatusChoiceProps } from "@/types/index";
import { BaseModel } from "@/types/organization";

export interface QualifierCode extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  code: string;
  description: string;
}

export type QualifierCodeFormValues = Omit<
  QualifierCode,
  "id" | "organization" | "created" | "modified"
>;

export type StopTypeProps = "P" | "SP" | "SD" | "D" | "DO";

export interface Stop extends BaseModel {
  id: string;
  status: ShipmentStatusChoiceProps;
  sequence?: number | undefined;
  movement: string;
  location?: string;
  pieces?: number;
  weight?: string;
  addressLine?: string;
  appointmentTimeWindowStart: string;
  appointmentTimeWindowEnd: string;
  arrivalTime?: string;
  departureTime?: string;
  stopType: StopTypeProps;
  stopComments?: StopCommentFormValues[];
}

export type StopFormValues = Omit<
  Stop,
  "id" | "organization" | "created" | "modified" | "movement"
>;

export interface StopComment extends BaseModel {
  id: string;
  qualifierCode: string;
  value: string;
}

export type StopCommentFormValues = Omit<
  StopComment,
  "id" | "organization" | "created" | "modified"
>;

export interface ServiceIncident extends BaseModel {
  id: string;
  movement: string;
  stop: string;
  delayCode?: string;
  delayReason?: string;
  delayTime?: any;
}
