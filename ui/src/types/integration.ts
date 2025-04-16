export const enum IntegrationType {
  GoogleMaps = "GoogleMaps",
  PCMiler = "PCMiler",
}

export const enum IntegrationCategory {
  MappingRouting = "MappingRouting",
  FreightLogistics = "FreightLogistics",
  Telematics = "Telematics",
}

export type Integration = {
  id: string;
  type: IntegrationType;
  name: string;
  category: IntegrationCategory;
  builtBy: string;
  description: string;
  enabled: boolean;
  configuration?: Record<string, any>;
  createdAt: number;
  updatedAt: number;
};

export type GoogleMapsConfigData = {
  apiKey: string;
};

export type PCMilerConfigData = {
  username: string;
  password: string;
  licenseKey: string;
};
