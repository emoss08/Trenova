/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { IntegrationCategory, IntegrationType } from "@/types/integration";

export const integrationImages: Record<IntegrationType, string> = {
  [IntegrationType.GoogleMaps]:
    "https://raw.githubusercontent.com/gilbarbara/logos/refs/heads/main/logos/google-maps.svg",
  [IntegrationType.PCMiler]:
    "https://www.pcmiler.com/img/alk-logos/pcmiler-logo.svg",
};

// External docs for each integration type
export const integrationDocs: Record<IntegrationType, string> = {
  [IntegrationType.GoogleMaps]:
    "https://developers.google.com/maps/documentation",
  [IntegrationType.PCMiler]: "https://developer.trimblemaps.com/",
};

// Which integrations should show a Featured ribbon
export const featuredIntegrations: Record<IntegrationType, boolean> = {
  [IntegrationType.GoogleMaps]: true,
  [IntegrationType.PCMiler]: false,
};

// Helper function to get a human-readable category name
export const getCategoryDisplayName = (
  category: IntegrationCategory,
): string => {
  switch (category) {
    case IntegrationCategory.MappingRouting:
      return "Mapping & Routing";
    case IntegrationCategory.FreightLogistics:
      return "Freight Logistics";
    default:
      return category;
  }
};
