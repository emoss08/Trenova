/**
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
