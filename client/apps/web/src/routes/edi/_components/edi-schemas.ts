import { translate } from "@trenova/shared/i18n/runtime";
import { timezoneChoices } from "@/lib/choices";
import {
  ediConnectionMethodSchema,
  ediMappingEntityTypeSchema,
  ediPartnerKindSchema,
  type EDIPartner,
  type UpsertEDIPartnerRequest,
} from "@trenova/shared/types/edi";
import type { SELECT_OPTIONS_ENDPOINTS } from "@trenova/shared/types/server";
import { z } from "zod";

export const mappingEntityTypes = ediMappingEntityTypeSchema.options;

export const mappingTargetEndpoints: Record<
  (typeof mappingEntityTypes)[number],
  SELECT_OPTIONS_ENDPOINTS
> = {
  Customer: "/customers/select-options/",
  ServiceType: "/service-types/select-options/",
  ShipmentType: "/shipment-types/select-options/",
  FormulaTemplate: "/formula-templates/select-options/",
  Location: "/locations/select-options/",
  Commodity: "/commodities/select-options/",
  AccessorialCharge: "/accessorial-charges/select-options/",
  ServiceFailureReasonCode: "/service-failure-reason-codes/select-options/",
};

export const communicationProfileMethods = ediConnectionMethodSchema.options;

export type CommunicationProfileMethod = (typeof communicationProfileMethods)[number];

export const communicationProfileMethodOptions = communicationProfileMethods.map((method) => ({
  label: method,
  value: method,
}));

export const profileStatusOptions = [
  { label: "Active", value: "Active" },
  { label: "Inactive", value: "Inactive" },
];

export const partnerCountryOptions = [
  { label: "United States", value: "US" },
  { label: "Canada", value: "CA" },
  { label: "Mexico", value: "MX" },
];

export const partnerTimezoneOptions = timezoneChoices;

export const mdnModeOptions = [
  { label: "Synchronous", value: "sync" },
  { label: "Asynchronous", value: "async" },
];

export const as2SigningAlgorithmOptions = [
  { label: "SHA-256", value: "sha256" },
  { label: "SHA-384", value: "sha384" },
  { label: "SHA-512", value: "sha512" },
  { label: "SHA-1 (legacy)", value: "sha1" },
];

export const as2EncryptionAlgorithmOptions = [
  { label: "AES-256-CBC", value: "aes256-cbc" },
  { label: "AES-128-CBC", value: "aes128-cbc" },
  { label: "AES-256-GCM", value: "aes256-gcm" },
  { label: "AES-128-GCM", value: "aes128-gcm" },
  { label: "Triple DES (legacy)", value: "3des" },
];

export const as2InboundRequirementOptions = [
  { label: "Automatic (require when certificates are configured)", value: "auto" },
  { label: "Required", value: "true" },
  { label: "Not required", value: "false" },
];

export const as2CompressionOptions = [
  { label: "None", value: "none" },
  { label: "Zlib", value: "zlib" },
];

export const environmentOptions = [
  { label: "Test", value: "test" },
  { label: "Production", value: "production" },
];

export const sftpAuthModeOptions = [
  { label: "Private key", value: "privateKey" },
  { label: "Password", value: "password" },
];

export const communicationProfileConfigSchema = z.object({
  connectedOrganizationId: z.string(),
  localAS2Id: z.string(),
  partnerAS2Id: z.string(),
  endpointUrl: z.string(),
  localCertificate: z.string(),
  partnerSigningCertificate: z.string(),
  partnerEncryptionCertificate: z.string(),
  mdnMode: z.string(),
  mdnUrl: z.string(),
  compressionAlgorithm: z.string(),
  signingAlgorithm: z.string(),
  encryptionAlgorithm: z.string(),
  requireSignedInbound: z.string(),
  requireEncryptedInbound: z.string(),
  basicAuthUsername: z.string(),
  retryMaxAttempts: z.string(),
  retryInitialIntervalSeconds: z.string(),
  retryMaxIntervalSeconds: z.string(),
  host: z.string(),
  port: z.string(),
  username: z.string(),
  authMode: z.string(),
  inboundDirectory: z.string(),
  outboundDirectory: z.string(),
  archiveDirectory: z.string(),
  fileNamingPattern: z.string(),
  knownHostKey: z.string(),
  providerName: z.string(),
  mailboxId: z.string(),
  accountId: z.string(),
  senderMailboxId: z.string(),
  receiverMailboxId: z.string(),
  isaRoutingId: z.string(),
  gsRoutingId: z.string(),
  endpoint: z.string(),
  contactEmail: z.string(),
  environment: z.string(),
});

export const communicationProfileSecretsSchema = z.object({
  basicAuthPassword: z.string(),
  password: z.string(),
  privateKey: z.string(),
  credential: z.string(),
  token: z.string(),
});

export const communicationProfileFormSchema = z.object({
  ediConnectionId: z.string(),
  ediPartnerId: z.string(),
  method: ediConnectionMethodSchema,
  status: z.string().min(1, { error: () => translate("Status is required") }),
  name: z.string().min(1, { error: () => translate("Name is required") }),
  description: z.string(),
  config: communicationProfileConfigSchema,
  secrets: communicationProfileSecretsSchema,
});

export type CommunicationProfileFormValues = z.infer<typeof communicationProfileFormSchema>;

export const createInternalPartnerPairSchema = z.object({
  targetOrganizationId: z
    .string()
    .min(1, { error: () => translate("Target organization is required") }),
  sourceCode: z.string().min(1, { error: () => translate("Source partner code is required") }),
  sourceName: z.string().min(1, { error: () => translate("Source partner name is required") }),
  sourceDescription: z.string(),
  sourceContactName: z.string(),
  sourceContactEmail: z.string(),
  sourceContactPhone: z.string(),
  sourceEnabledForInbound: z.boolean(),
  sourceEnabledForOutbound: z.boolean(),
  sourceSettings: z.record(z.string(), z.unknown()),
  targetCode: z.string().min(1, { error: () => translate("Target partner code is required") }),
  targetName: z.string().min(1, { error: () => translate("Target partner name is required") }),
  targetDescription: z.string(),
  targetContactName: z.string(),
  targetContactEmail: z.string(),
  targetContactPhone: z.string(),
  targetEnabledForInbound: z.boolean(),
  targetEnabledForOutbound: z.boolean(),
  targetSettings: z.record(z.string(), z.unknown()),
});

export type CreateInternalPartnerPairFormValues = z.infer<typeof createInternalPartnerPairSchema>;

export const ediPartnerFormSchema = z.object({
  kind: ediPartnerKindSchema,
  status: z.string().min(1, { error: () => translate("Status is required") }),
  code: z.string().min(1, { error: () => translate("Partner code is required") }),
  name: z.string().min(1, { error: () => translate("Partner name is required") }),
  description: z.string(),
  internalOrganizationId: z.string(),
  ediConnectionId: z.string(),
  customerId: z.string(),
  country: z
    .string()
    .min(2, { error: () => translate("Country is required") })
    .max(2),
  timezone: z.string(),
  contactName: z.string(),
  contactEmail: z.string(),
  contactPhone: z.string(),
  enabledForInbound: z.boolean(),
  enabledForOutbound: z.boolean(),
  defaultTransportId: z.string(),
  defaultMappingProfileId: z.string(),
  settingsJson: z.string().refine(
    (value) => {
      try {
        const parsed = JSON.parse(value);
        return parsed !== null && !Array.isArray(parsed) && typeof parsed === "object";
      } catch {
        return false;
      }
    },
    { error: () => translate("Settings must be a valid JSON object") },
  ),
  version: z.number().optional(),
});

export type EDIPartnerFormValues = z.infer<typeof ediPartnerFormSchema>;

export function getPartnerFormDefaults(partner?: EDIPartner | null): EDIPartnerFormValues {
  return {
    kind: partner?.kind ?? "External",
    status: partner?.status ?? "Active",
    code: partner?.code ?? "",
    name: partner?.name ?? "",
    description: partner?.description ?? "",
    internalOrganizationId: partner?.internalOrganizationId ?? "",
    ediConnectionId: partner?.ediConnectionId ?? "",
    customerId: partner?.customerId ?? "",
    country: partner?.country ?? "US",
    timezone: partner?.timezone ?? "",
    contactName: partner?.contactName ?? "",
    contactEmail: partner?.contactEmail ?? "",
    contactPhone: partner?.contactPhone ?? "",
    enabledForInbound: partner?.enabledForInbound ?? true,
    enabledForOutbound: partner?.enabledForOutbound ?? true,
    defaultTransportId: partner?.defaultTransportId ?? "",
    defaultMappingProfileId: partner?.defaultMappingProfileId ?? "",
    settingsJson: JSON.stringify(partner?.settings ?? {}, null, 2),
    version: partner?.version,
  };
}

export function toPartnerRequest(values: EDIPartnerFormValues): UpsertEDIPartnerRequest {
  const request: UpsertEDIPartnerRequest = {
    kind: values.kind,
    status: values.status,
    code: values.code.trim(),
    name: values.name.trim(),
    description: emptyToUndefined(values.description),
    customerId: emptyToUndefined(values.customerId),
    defaultTransportId: emptyToUndefined(values.defaultTransportId),
    defaultMappingProfileId: emptyToUndefined(values.defaultMappingProfileId),
    country: values.country,
    timezone: emptyToUndefined(values.timezone),
    contactName: emptyToUndefined(values.contactName),
    contactEmail: emptyToUndefined(values.contactEmail),
    contactPhone: emptyToUndefined(values.contactPhone),
    enabledForInbound: values.enabledForInbound,
    enabledForOutbound: values.enabledForOutbound,
    settings: JSON.parse(values.settingsJson) as Record<string, unknown>,
    version: values.version,
  };

  if (values.kind === "Internal") {
    request.internalOrganizationId = emptyToUndefined(values.internalOrganizationId);
    request.ediConnectionId = emptyToUndefined(values.ediConnectionId);
  }

  return request;
}

export function emptyToUndefined(value: string) {
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}
