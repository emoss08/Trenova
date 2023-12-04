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

import { StatusChoiceProps } from "@/types/index";
import { BaseModel } from "@/types/organization";

export interface LocationCategory extends BaseModel {
  id: string;
  name: string;
  description?: string | null;
  color?: string | null;
}

export type LocationCategoryFormValues = Omit<
  LocationCategory,
  "organization" | "created" | "modified" | "id"
>;

export interface LocationComment extends BaseModel {
  id: string;
  location: string;
  commentType: string;
  commentTypeName: string;
  comment: string;
  enteredBy: string;
  enteredByUsername: string;
}

export type LocationCommentFormValues = Omit<
  LocationComment,
  | "organization"
  | "created"
  | "modified"
  | "id"
  | "location"
  | "enteredBy"
  | "commentTypeName"
  | "enteredByUsername"
>;

export interface LocationContact extends BaseModel {
  id: string;
  location: string;
  name: string;
  email?: string | null;
  phone?: string | null;
  fax?: string | null;
}

export type LocationContactFormValues = Omit<
  LocationContact,
  "organization" | "created" | "modified" | "id" | "location"
>;

export interface Location extends BaseModel {
  id: string;
  name: string;
  code: string;
  status: StatusChoiceProps;
  locationCategory?: string | null;
  depot?: string | null;
  description?: string | null;
  addressLine1: string;
  addressLine2?: string | null;
  city: string;
  state: string;
  zipCode: string;
  longitude?: number | null;
  latitude?: number | null;
  placeId?: string;
  isGeocoded: boolean;
  locationColor?: string | null;
  locationCategoryName?: string | null;
  pickupCount: number;
  waitTimeAvg: number;
  locationComments: LocationComment[];
  locationContacts: LocationContact[];
}

export type LocationFormValues = Omit<
  Location,
  | "organization"
  | "id"
  | "longitude"
  | "latitude"
  | "locationColor"
  | "locationCategoryName"
  | "pickupCount"
  | "waitTimeAvg"
  | "locationContacts"
  | "locationComments"
  | "isGeocoded"
  | "placeId"
  | "created"
  | "modified"
> & {
  locationComments?: LocationCommentFormValues[] | null;
  locationContacts?: LocationContactFormValues[] | null;
};
