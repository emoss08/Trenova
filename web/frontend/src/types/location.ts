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

import { type StatusChoiceProps } from "@/types/index";
import { type BaseModel } from "@/types/organization";
import { type User } from "./accounts";
import { type CommentType } from "./dispatch";

export interface LocationCategory extends BaseModel {
  id: string;
  name: string;
  description?: string;
  color?: string;
}

export type LocationCategoryFormValues = Omit<
  LocationCategory,
  "organizationId" | "createdAt" | "updatedAt" | "id" | "version"
>;

export interface LocationComment extends BaseModel {
  id: string;
  locationId: string;
  // Comment Type ID.
  commentTypeId: string;
  // The actual comment.
  comment: string;
  // User that entered the comment.
  userId: string;
  edges?: {
    user: User;
    commentType: CommentType;
  };
}

export type LocationCommentFormValues = Omit<
  LocationComment,
  | "organizationId"
  | "createdAt"
  | "updatedAt"
  | "id"
  | "locationId"
  | "edges"
  | "version"
>;

export interface LocationContact extends BaseModel {
  id: string;
  location: string;
  name: string;
  emailAddress?: string;
  phoneNumber?: string;
}

export type LocationContactFormValues = Omit<
  LocationContact,
  "organizationId" | "createdAt" | "updatedAt" | "id" | "location" | "version"
>;

export interface Location extends BaseModel {
  id: string;
  name: string;
  code?: string;
  status: StatusChoiceProps;
  locationCategoryId?: string | null;
  description?: string;
  addressLine1: string;
  addressLine2?: string;
  city: string;
  stateId: string;
  postalCode: string;
  longitude?: number;
  latitude?: number;
  placeId?: string;
  isGeocoded: boolean;
  locationCategory?: LocationCategory;
  state?: USStates;
  comments: LocationComment[];
  contacts: LocationContact[];
}

export type LocationFormValues = Omit<
  Location,
  | "organizationId"
  | "id"
  | "longitude"
  | "latitude"
  | "isGeocoded"
  | "placeId"
  | "createdAt"
  | "updatedAt"
  | "locationCategory"
  | "state"
  | "comments"
  | "contacts"
  | "version"
> & {
  comments?: LocationCommentFormValues[] | null;
  contacts?: LocationContactFormValues[] | null;
};

export type USStates = {
  id: string;
  name: string;
  abbreviation: string;
  countryName: string;
  countryIso3: string;
};

export type GoogleAutoCompleteResult = {
  address: string;
  name: string;
  placeId: string;
};

export type MonthlyPickupData = {
  name: string;
  total: number;
};
