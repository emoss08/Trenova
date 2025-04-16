export const enum IntegrationType {
  GoogleMaps = "GoogleMaps",
  PCMiler = "PCMiler",
}

export const enum IntegrationCategory {
  MappingRouting = "MappingRouting",
  FreightLogistics = "FreightLogistics",
}

export const enum FieldType {
  Text = "text",
  Password = "password",
  Select = "select",
  Toggle = "toggle",
  Textarea = "textarea",
  Email = "email",
  Number = "number",
  Url = "url",
}

export type FieldOption = {
  label: string;
  value: string;
};

export type FieldValidation = {
  pattern?: string;
  min?: number;
  max?: number;
  minLength?: number;
  maxLength?: number;
  message?: string;
};

export type Field = {
  key: string;
  name: string;
  description: string;
  type: FieldType;
  required: boolean;
  defaultValue?: any;
  options?: FieldOption[];
  validation?: FieldValidation;
  placeholder?: string;
  group?: string;
  order: number;
};

export type TriggerEvent =
  | "issue_created"
  | "issue_updated"
  | "issue_commented"
  | "issue_status_changed"
  | "issue_assigned";

export type EventTrigger = {
  event: TriggerEvent;
  description: string;
  enabled: boolean;
  requiredFields?: string[];
};

export type WebhookEndpoint = {
  name: string;
  url: string;
  description: string;
  enabled: boolean;
  secret?: string;
  events: TriggerEvent[];
};

export type Integration = {
  id: string;
  type: IntegrationType;
  name: string;
  category: IntegrationCategory;
  builtBy: string;
  description: string;
  enabled: boolean;
  overview?: string;
  screenshots?: string[];
  features?: string[];
  configFields?: Record<string, Field>;
  eventTriggers?: EventTrigger[];
  webhookEndpoints?: WebhookEndpoint[];
  configuration?: Record<string, any>;
  lastUsed: number;
  usageCount: number;
  errorCount: number;
  lastError: string;
  lastErrorAt: number;
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
