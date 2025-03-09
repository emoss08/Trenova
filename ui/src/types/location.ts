import { LocationCategorySchema } from "@/lib/schemas/location-category-schema";
import { LocationSchema } from "@/lib/schemas/location-schema";

export type Location = {
  locationCategory?: LocationCategorySchema | null;
} & LocationSchema;
