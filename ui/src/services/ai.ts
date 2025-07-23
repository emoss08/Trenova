/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { http } from "@/lib/http-client";

export type ClassificationRequest = {
  name: string;
  description?: string;
  address?: string;
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
      "/ai/classify/location",
      req,
    );
    return response.data;
  }
}

export const aiAPI = new AIAPI();
