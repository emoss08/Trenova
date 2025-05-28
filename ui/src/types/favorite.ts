import { type BaseModel } from "./common";

export interface Favorite extends BaseModel {
  // Primary identifiers
  businessUnitId: string;
  organizationId: string;
  userId: string;

  // Core fields
  pageUrl: string;
  pageTitle: string;
  pageSection?: string;
  icon?: string;
  description?: string;
}

export interface CreateFavoriteRequest {
  pageUrl: string;
  pageTitle: string;
  pageSection?: string;
  icon?: string;
  description?: string;
}

export interface UpdateFavoriteRequest {
  pageTitle: string;
  pageSection?: string;
  icon?: string;
  description?: string;
}

export interface ToggleFavoriteRequest {
  pageUrl: string;
  pageTitle: string;
  pageSection?: string;
  icon?: string;
  description?: string;
}

export interface ToggleFavoriteResponse {
  action: "added" | "removed";
  favorite: Favorite | null;
}

export interface CheckFavoriteResponse {
  isFavorite: boolean;
  favorite: Favorite | null;
}