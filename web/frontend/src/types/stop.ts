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

import { type ShipmentStatusChoiceProps } from "@/lib/choices";
import { type StatusChoiceProps } from "@/types/index";
import { type BaseModel } from "@/types/organization";

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
  "id" | "organizationId" | "createdAt" | "updatedAt" | "movement" | "version"
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
