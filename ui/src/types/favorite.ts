/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

// TODO(wolfred): Convert to zod schema
export interface Favorite {
  id: string;
  version: number;
  createdAt: string;
  updatedAt: string;
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
