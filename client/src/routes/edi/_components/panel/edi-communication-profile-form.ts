import type { EDICommunicationProfile, UpsertEDICommunicationProfileRequest } from "@/types/edi";
import {
  emptyToUndefined,
  type CommunicationProfileFormValues,
  type CommunicationProfileMethod,
} from "../edi-schemas";

type ProfileConfigKey = keyof CommunicationProfileFormValues["config"];

const x12EnvelopeKeys: ProfileConfigKey[] = [
  "isaSenderQualifier",
  "isaSenderId",
  "isaReceiverQualifier",
  "isaReceiverId",
  "gsSenderId",
  "gsReceiverId",
  "x12Version",
  "environment",
  "acknowledgmentPreference",
];

const sftpEndpointKeys: ProfileConfigKey[] = [
  "host",
  "port",
  "username",
  "authMode",
  "inboundDirectory",
  "outboundDirectory",
  "archiveDirectory",
  "fileNamingPattern",
  "knownHostKey",
];

const configKeysByMethod: Record<CommunicationProfileMethod, ProfileConfigKey[]> = {
  Internal: ["connectedOrganizationId"],
  AS2: [
    "localAS2Id",
    "partnerAS2Id",
    "endpointUrl",
    "localCertificate",
    "partnerSigningCertificate",
    "partnerEncryptionCertificate",
    "mdnMode",
    "mdnUrl",
    "compressionAlgorithm",
    "signingAlgorithm",
    "encryptionAlgorithm",
    "requireSignedInbound",
    "requireEncryptedInbound",
    "basicAuthUsername",
    ...x12EnvelopeKeys,
  ],
  SFTP: [...sftpEndpointKeys, ...x12EnvelopeKeys],
  VAN: [
    "providerName",
    "mailboxId",
    "accountId",
    "contactEmail",
    ...sftpEndpointKeys,
    ...x12EnvelopeKeys,
  ],
};

export function getProfileFormDefaults(
  profile: EDICommunicationProfile | null,
): CommunicationProfileFormValues {
  const config = profile?.config ?? {};
  return {
    ediConnectionId: profile?.ediConnectionId ?? "",
    ediPartnerId: profile?.ediPartnerId ?? "",
    method: (profile?.method ?? "Internal") as CommunicationProfileMethod,
    status: profile?.status ?? "Active",
    name: profile?.name ?? "",
    description: profile?.description ?? "",
    config: {
      connectedOrganizationId: stringConfig(config, "connectedOrganizationId"),
      localAS2Id: stringConfig(config, "localAS2Id"),
      partnerAS2Id: stringConfig(config, "partnerAS2Id"),
      endpointUrl: stringConfig(config, "endpointUrl"),
      localCertificate: stringConfig(config, "localCertificate"),
      partnerSigningCertificate: stringConfig(config, "partnerSigningCertificate"),
      partnerEncryptionCertificate: stringConfig(config, "partnerEncryptionCertificate"),
      mdnMode: stringConfig(config, "mdnMode", "sync"),
      mdnUrl: stringConfig(config, "mdnUrl"),
      compressionAlgorithm: stringConfig(config, "compressionAlgorithm", "none"),
      signingAlgorithm: stringConfig(config, "signingAlgorithm", "sha256"),
      encryptionAlgorithm: stringConfig(config, "encryptionAlgorithm", "aes256-cbc"),
      requireSignedInbound: stringConfig(config, "requireSignedInbound", "auto"),
      requireEncryptedInbound: stringConfig(config, "requireEncryptedInbound", "auto"),
      basicAuthUsername: stringConfig(config, "basicAuthUsername"),
      host: stringConfig(config, "host"),
      port: stringConfig(config, "port", "22"),
      username: stringConfig(config, "username"),
      authMode: stringConfig(config, "authMode", "privateKey"),
      inboundDirectory: stringConfig(config, "inboundDirectory"),
      outboundDirectory: stringConfig(config, "outboundDirectory"),
      archiveDirectory: stringConfig(config, "archiveDirectory"),
      fileNamingPattern: stringConfig(config, "fileNamingPattern", "{partner}-{timestamp}.edi"),
      knownHostKey: stringConfig(config, "knownHostKey"),
      providerName: stringConfig(config, "providerName"),
      mailboxId: stringConfig(config, "mailboxId"),
      accountId: stringConfig(config, "accountId"),
      senderMailboxId: stringConfig(config, "senderMailboxId"),
      receiverMailboxId: stringConfig(config, "receiverMailboxId"),
      isaRoutingId: stringConfig(config, "isaRoutingId"),
      gsRoutingId: stringConfig(config, "gsRoutingId"),
      endpoint: stringConfig(config, "endpoint"),
      contactEmail: stringConfig(config, "contactEmail"),
      isaSenderQualifier: stringConfig(config, "isaSenderQualifier"),
      isaSenderId: stringConfig(config, "isaSenderId"),
      isaReceiverQualifier: stringConfig(config, "isaReceiverQualifier"),
      isaReceiverId: stringConfig(config, "isaReceiverId"),
      gsSenderId: stringConfig(config, "gsSenderId"),
      gsReceiverId: stringConfig(config, "gsReceiverId"),
      x12Version: stringConfig(config, "x12Version", "004010"),
      environment: stringConfig(config, "environment", "test"),
      acknowledgmentPreference: stringConfig(config, "acknowledgmentPreference", "997"),
    },
    secrets: {
      basicAuthPassword: "",
      password: "",
      privateKey: "",
      credential: "",
      token: "",
    },
  };
}

export function toCommunicationProfileRequest(
  values: CommunicationProfileFormValues,
  profile: EDICommunicationProfile | null,
): UpsertEDICommunicationProfileRequest {
  return {
    ediConnectionId: emptyToUndefined(values.ediConnectionId),
    ediPartnerId: emptyToUndefined(values.ediPartnerId),
    method: values.method,
    status: values.status,
    name: values.name,
    description: values.description,
    config: compactProfileConfig(values.method, values.config),
    secrets: compactSecrets(values.secrets),
    version: profile?.version,
  };
}

function compactProfileConfig(
  method: CommunicationProfileMethod,
  config: CommunicationProfileFormValues["config"],
): Record<string, unknown> {
  return Object.fromEntries(
    configKeysByMethod[method]
      .map((key) => [key, config[key]] as const)
      .filter(([, value]) => String(value ?? "").trim() !== ""),
  );
}

function compactSecrets(
  secrets: CommunicationProfileFormValues["secrets"],
): Record<string, string> {
  return Object.fromEntries(Object.entries(secrets).filter(([, value]) => value.trim() !== ""));
}

function stringConfig(config: Record<string, unknown>, key: string, fallback = "") {
  const value = config[key];
  if (value === undefined || value === null) return fallback;
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return value.toString();
  return fallback;
}
