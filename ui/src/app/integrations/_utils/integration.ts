import {
  IntegrationCategory,
  IntegrationType,
} from "@/types/integrations/integration";

export const integrationImages: Record<IntegrationType, string> = {
  [IntegrationType.GoogleMaps]:
    "https://raw.githubusercontent.com/gilbarbara/logos/refs/heads/main/logos/google-maps.svg",
  [IntegrationType.PCMiler]:
    "https://www.pcmiler.com/img/alk-logos/pcmiler-logo.svg",
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
