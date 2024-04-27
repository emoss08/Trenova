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
  code: string;
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
  edges?: {
    locationCategory?: LocationCategory;
    state?: USStates;
    comments: LocationComment[];
    contacts: LocationContact[];
  };
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
  | "edges"
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
