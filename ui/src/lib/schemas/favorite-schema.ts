import { z } from "zod";

export const createFavoriteSchema = z.object({
  pageUrl: z
    .string()
    .url("Invalid URL format")
    .min(1, "Page URL is required")
    .max(500, "Page URL must be 500 characters or less"),
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
    .max(50, "Icon must be 50 characters or less")
    .optional(),
  description: z
    .string()
    .max(1000, "Description must be 1000 characters or less")
    .optional(),
});

export type CreateFavoriteSchema = z.infer<typeof createFavoriteSchema>;

export const updateFavoriteSchema = z.object({
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
    .max(50, "Icon must be 50 characters or less")
    .optional(),
  description: z
    .string()
    .max(1000, "Description must be 1000 characters or less")
    .optional(),
});

export type UpdateFavoriteSchema = z.infer<typeof updateFavoriteSchema>;

export const toggleFavoriteSchema = z.object({
  pageUrl: z
    .string()
    .url("Invalid URL format")
    .min(1, "Page URL is required")
    .max(500, "Page URL must be 500 characters or less"),
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
    .max(50, "Icon must be 50 characters or less")
    .optional(),
  description: z
    .string()
    .max(1000, "Description must be 1000 characters or less")
    .optional(),
});

export type ToggleFavoriteSchema = z.infer<typeof toggleFavoriteSchema>;