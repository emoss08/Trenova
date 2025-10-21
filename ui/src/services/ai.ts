import { http } from "@/lib/http-client";

export type ClassificationRequest = {
  name: string;
  description?: string;
  address?: string;
  city?: string;
  state?: string;
  postalCode?: string;
  code?: string;
  placeId?: string;
  latitude?: number;
  longitude?: number;
};

export type ClassificationResponse = {
  category: string;
  categoryId: string;
  facilityType?: string;
  confidence: number;
  reasoning: string;
  alternativeCategories: {
    category: string;
    categoryId: string;
    confidence: number;
  }[];
};

export class AIAPI {
  async classifyLocation(req: ClassificationRequest) {
    const response = await http.post<ClassificationResponse>(
      "/classification/location/",
      req,
    );
    return response.data;
  }
}

export const aiAPI = new AIAPI();
