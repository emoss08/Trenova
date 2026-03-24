import { z } from "zod";

export const pageFavoriteSchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  userId: z.string(),
  pageUrl: z.string(),
  pageTitle: z.string(),
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
});

export type PageFavorite = z.infer<typeof pageFavoriteSchema>;

export const toggleFavoriteResponseSchema = z.object({
  favorited: z.boolean(),
  favorite: pageFavoriteSchema.nullish(),
});

export const toggleFavoriteRequestSchema = z.object({
  pageUrl: z.string(),
  pageTitle: z.string(),
});

export type ToggleFavoriteRequest = z.infer<typeof toggleFavoriteRequestSchema>;

export type ToggleFavoriteResponse = z.infer<
  typeof toggleFavoriteResponseSchema
>;

export const checkFavoriteResponseSchema = z.object({
  favorited: z.boolean(),
});

export type CheckFavoriteResponse = z.infer<typeof checkFavoriteResponseSchema>;
