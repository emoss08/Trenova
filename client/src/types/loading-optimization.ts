import { z } from "zod";

export const commodityPlacementSchema = z.object({
  commodityId: z.string(),
  commodityName: z.string(),
  positionFeet: z.number(),
  lengthFeet: z.number(),
  weight: z.number(),
  pieces: z.number(),
  stackable: z.boolean(),
  fragile: z.boolean(),
  isHazmat: z.boolean(),
  hazmatClass: z.string().optional(),
  minTemp: z.number().nullable().optional(),
  maxTemp: z.number().nullable().optional(),
  loadingInstructions: z.string().optional(),
  estimatedLength: z.boolean(),
  stopNumber: z.number().optional(),
});

export type CommodityPlacement = z.infer<typeof commodityPlacementSchema>;

export const hazmatZoneResultSchema = z.object({
  commodityAId: z.string(),
  commodityBId: z.string(),
  commodityAName: z.string(),
  commodityBName: z.string(),
  ruleName: z.string(),
  segregationType: z.string(),
  requiredDistanceFeet: z.number().nullable().optional(),
  actualDistanceFeet: z.number(),
  satisfied: z.boolean(),
});

export type HazmatZoneResult = z.infer<typeof hazmatZoneResultSchema>;

export const loadingWarningSchema = z.object({
  type: z.string(),
  message: z.string(),
  severity: z.enum(["error", "warning", "info"]),
  commodityIds: z.array(z.string()).optional(),
});

export type LoadingWarning = z.infer<typeof loadingWarningSchema>;

export const axleWeightSchema = z.object({
  axle: z.string(),
  weight: z.number(),
  limit: z.number(),
  percentage: z.number(),
  compliant: z.boolean(),
});

export type AxleWeight = z.infer<typeof axleWeightSchema>;

export const loadingRecommendationSchema = z.object({
  type: z.string(),
  priority: z.enum(["critical", "suggested", "optimization"]),
  title: z.string(),
  description: z.string(),
  impact: z.string().optional(),
  commodityIds: z.array(z.string()).optional(),
});

export type LoadingRecommendation = z.infer<typeof loadingRecommendationSchema>;

export const stopDividerSchema = z.object({
  positionFeet: z.number(),
  stopNumber: z.number(),
  label: z.string(),
});

export type StopDivider = z.infer<typeof stopDividerSchema>;

export const loadingOptimizationResultSchema = z.object({
  trailerLengthFeet: z.number(),
  totalLinearFeet: z.number(),
  totalWeight: z.number(),
  maxWeight: z.number(),
  linearFeetUtil: z.number(),
  weightUtil: z.number(),
  utilizationScore: z.number(),
  utilizationGrade: z.string(),
  placements: z.array(commodityPlacementSchema),
  hazmatZones: z.array(hazmatZoneResultSchema),
  warnings: z.array(loadingWarningSchema),
  axleWeights: z.array(axleWeightSchema),
  recommendations: z.array(loadingRecommendationSchema),
  stopDividers: z.array(stopDividerSchema).optional(),
  aiAnalysis: z.string().optional(),
});

export type LoadingOptimizationResult = z.infer<typeof loadingOptimizationResultSchema>;

export interface StopInput {
  sequence: number;
  locationName: string;
  locationCity: string;
}

export interface LoadingOptimizationRequest {
  commodities: {
    commodityId: string;
    pieces: number;
    weight: number;
  }[];
  equipmentTypeId?: string;
  stops?: StopInput[];
}
