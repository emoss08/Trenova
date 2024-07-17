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

import type {
  HazardousClassChoiceProps,
  PackingGroupChoiceProps,
  UnitOfMeasureChoiceProps,
} from "@/lib/choices";
import { type StatusChoiceProps } from "@/types/index";
import { type BaseModel } from "@/types/organization";

export interface HazardousMaterial extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  name: string;
  description?: string;
  hazardClass: HazardousClassChoiceProps;
  packingGroup?: PackingGroupChoiceProps;
  ergNumber?: string;
  properShippingName?: string;
}

export type HazardousMaterialFormValues = Omit<
  HazardousMaterial,
  "id" | "organizationId" | "createdAt" | "updatedAt" | "version"
>;

export interface Commodity extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  name: string;
  description?: string;
  minTemp?: number;
  maxTemp?: number;
  unitOfMeasure?: UnitOfMeasureChoiceProps;
  hazardousMaterialId?: string | null;
  isHazmat: boolean;
}

export type CommodityFormValues = Omit<
  Commodity,
  "id" | "createdAt" | "updatedAt" | "organizationId" | "version"
>;
