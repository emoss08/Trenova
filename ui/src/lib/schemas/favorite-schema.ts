/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import * as z from "zod/v4";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const favoriteSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  userId: optionalStringSchema,
  pageUrl: z.string().url({ error: "Invalid URL format" }),
  pageTitle: z.string().min(1, { error: "Page title is required" }),
});

export const createFavoriteSchema = z.object({
  pageUrl: z
    .url({ error: "Invalid URL format" })
    .min(1, { error: "Page URL is required" })
    .max(500, { error: "Page URL must be 500 characters or less" }),
  pageTitle: z
    .string()
    .min(1, { error: "Page title is required" })
    .max(255, { error: "Page title must be 255 characters or less" }),
  pageSection: z
    .string()
    .max(100, { error: "Page section must be 100 characters or less" })
    .optional(),
  icon: z
    .string()
    .max(50, { error: "Icon must be 50 characters or less" })
    .optional(),
  description: z
    .string()
    .max(1000, { error: "Description must be 1000 characters or less" })
    .optional(),
});

export const updateFavoriteSchema = z.object({
  pageTitle: z
    .string()
    .min(1, { error: "Page title is required" })
    .max(255, { error: "Page title must be 255 characters or less" }),
  pageSection: z
    .string()
    .max(100, { error: "Page section must be 100 characters or less" })
    .optional(),
  icon: z
    .string()
    .max(50, { error: "Icon must be 50 characters or less" })
    .optional(),
  description: z
    .string()
    .max(1000, { error: "Description must be 1000 characters or less" })
    .optional(),
});

export const toggleFavoriteSchema = z.object({
  pageUrl: z
    .url({ error: "Invalid URL format" })
    .min(1, { error: "Page URL is required" })
    .max(500, { error: "Page URL must be 500 characters or less" }),
  pageTitle: z
    .string()
    .min(1, "Page title is required")
    .max(255, "Page title must be 255 characters or less"),
  pageSection: z
    .string()
    .max(100, "Page section must be 100 characters or less")
    .optional(),
  icon: z
    .string()
    .max(50, { error: "Icon must be 50 characters or less" })
    .optional(),
  description: z
    .string()
    .max(1000, { error: "Description must be 1000 characters or less" })
    .optional(),
});

export type ToggleFavoriteSchema = z.infer<typeof toggleFavoriteSchema>;
export type FavoriteSchema = z.infer<typeof favoriteSchema>;
export type UpdateFavoriteSchema = z.infer<typeof updateFavoriteSchema>;
export type CreateFavoriteSchema = z.infer<typeof createFavoriteSchema>;
