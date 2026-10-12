import type {
  AIProviderKind,
  AIProviderPlatform,
  AIProviderPlatformAuth,
  AIProviderPlatformDescriptor,
  AIProviderPlatformSetting,
  AIProviderPlatformSettings,
} from "@/types/ai-provider";
import { defineLabels } from "@trenova/shared/i18n/labels";

/**
 * The platform settings as the editor holds them: every setting a string, so a
 * platform switch keeps what was typed for the one before without sending it.
 * The auth is empty until a platform is chosen.
 */
export type PlatformSettingsDraft = {
  auth: AIProviderPlatformAuth | "";
  region: string;
  projectId: string;
  workspaceId: string;
  accessKeyId: string;
  resource: string;
  tenantId: string;
  clientId: string;
};

export const EMPTY_PLATFORM_SETTINGS: PlatformSettingsDraft = {
  auth: "",
  region: "",
  projectId: "",
  workspaceId: "",
  accessKeyId: "",
  resource: "",
  tenantId: "",
  clientId: "",
};

export const PLATFORM_LABELS: Record<AIProviderPlatform, string> = defineLabels({
  Direct: "Direct",
  Bedrock: "Amazon Bedrock",
  VertexAI: "Google Vertex AI",
  ClaudePlatformAWS: "Claude Platform on AWS",
  ClaudePlatformGoogleCloud: "Claude Platform on Google Cloud",
  Foundry: "Microsoft Foundry",
});

export const PLATFORM_DESCRIPTIONS: Record<AIProviderPlatform, string> = defineLabels({
  Direct: "The base URL, with an API key.",
  Bedrock: "Claude through your AWS account's Bedrock.",
  VertexAI: "Claude as a Vertex AI model in your Google Cloud project.",
  ClaudePlatformAWS: "The Claude API, run by Anthropic and billed through AWS.",
  ClaudePlatformGoogleCloud: "The Claude API, run by Anthropic and billed through Google Cloud.",
  Foundry: "Claude deployed in your Microsoft Foundry resource.",
});

export const PLATFORM_AUTH_LABELS: Record<AIProviderPlatformAuth, string> = defineLabels({
  ApiKey: "API key",
  AccessKey: "IAM access key",
  ServiceAccount: "Service account key",
  EntraID: "Microsoft Entra ID",
  Server: "This server's credentials",
});

export const PLATFORM_AUTH_DESCRIPTIONS: Record<AIProviderPlatformAuth, string> = defineLabels({
  ApiKey: "A key the platform issued.",
  AccessKey: "An access key ID and its secret; each request is signed.",
  ServiceAccount: "A JSON key file for a service account allowed to call Claude.",
  EntraID: "An app registration's client ID and secret.",
  Server: "The identity of the server Trenova runs on, such as its IAM role. Nothing is stored.",
});

/** The label a credential field takes for the way a platform signs in. */
export const CREDENTIAL_LABELS: Record<AIProviderPlatformAuth, string> = defineLabels({
  ApiKey: "API key",
  AccessKey: "Secret access key",
  ServiceAccount: "Service account key (JSON)",
  EntraID: "Client secret",
  Server: "Sign-in credential",
});

export const PLATFORM_SETTING_LABELS: Record<AIProviderPlatformSetting, string> = defineLabels({
  region: "Region",
  projectId: "Project ID",
  workspaceId: "Workspace ID",
  resource: "Resource name",
  accessKeyId: "Access key ID",
  tenantId: "Tenant ID",
  clientId: "Client ID",
});

export const PLATFORM_SETTING_DESCRIPTIONS: Record<AIProviderPlatformSetting, string> =
  defineLabels({
    region: "Where the platform serves Claude for you.",
    projectId: "The Google Cloud project Claude is enabled in.",
    workspaceId: "The Anthropic workspace requests are made in.",
    resource: "The name before .services.ai.azure.com in its endpoint address.",
    accessKeyId: "The ID of the IAM access key, not its secret.",
    tenantId: "The Entra ID tenant ID, or its primary domain.",
    clientId: "The application (client) ID of the app registration.",
  });

const SETTING_ORDER: readonly AIProviderPlatformSetting[] = [
  "region",
  "projectId",
  "workspaceId",
  "resource",
  "accessKeyId",
  "tenantId",
  "clientId",
];

/** What the editor says when a setting the platform needs is blank. */
export const PLATFORM_SETTING_MISSING: Record<AIProviderPlatformSetting, string> = defineLabels({
  region: "Add the region",
  projectId: "Add the project ID",
  workspaceId: "Add the workspace ID",
  resource: "Add the resource name",
  accessKeyId: "Add the access key ID",
  tenantId: "Add the tenant ID",
  clientId: "Add the client ID",
});

/** Placeholders are examples of the shape each setting takes, not words. */
const SETTING_PLACEHOLDERS: Record<AIProviderPlatformSetting, string> = {
  region: "us-east-1",
  projectId: "my-project",
  workspaceId: "wrkspc_01…",
  resource: "my-foundry",
  accessKeyId: "AKIA…",
  tenantId: "contoso.onmicrosoft.com",
  clientId: "00000000-0000-0000-0000-000000000000",
};

const REGION_PLACEHOLDERS: Partial<Record<AIProviderPlatform, string>> = {
  VertexAI: "us-east5",
  ClaudePlatformGoogleCloud: "global",
};

export function settingPlaceholder(
  platform: AIProviderPlatform,
  setting: AIProviderPlatformSetting,
): string {
  if (setting === "region") {
    return REGION_PLACEHOLDERS[platform] ?? SETTING_PLACEHOLDERS.region;
  }
  return SETTING_PLACEHOLDERS[setting];
}

/** Whether a provider of this kind is reached through a platform at all. */
export function takesPlatform(kind: AIProviderKind): boolean {
  return kind === "AnthropicMessages";
}

/** The platform a provider is reached through: always Direct for another kind. */
export function effectivePlatform(
  kind: AIProviderKind,
  platform: AIProviderPlatform,
): AIProviderPlatform {
  return takesPlatform(kind) ? platform : "Direct";
}

export function platformDescriptor(
  platforms: readonly AIProviderPlatformDescriptor[],
  platform: AIProviderPlatform,
): AIProviderPlatformDescriptor | undefined {
  return platforms.find((entry) => entry.platform === platform);
}

/**
 * The ways a platform signs in that this editor offers. The server's own
 * credentials are offered only where the server allows them, unless the
 * provider already uses them, so an existing one stays as it was saved.
 */
export function offeredAuths(
  descriptor: AIProviderPlatformDescriptor | undefined,
  serverAllowed: boolean,
  current: AIProviderPlatformAuth | "",
): AIProviderPlatformAuth[] {
  if (!descriptor) return [];
  return descriptor.auths
    .map((entry) => entry.auth)
    .filter((auth) => auth !== "Server" || serverAllowed || current === "Server");
}

/** The way a platform signs in when the editor first opens on it. */
export function defaultAuth(
  descriptor: AIProviderPlatformDescriptor | undefined,
  serverAllowed: boolean,
): AIProviderPlatformAuth | "" {
  return offeredAuths(descriptor, serverAllowed, "")[0] ?? "";
}

/** The auth a draft uses: the one chosen if the platform takes it, else the platform's default. */
export function resolvedAuth(
  descriptor: AIProviderPlatformDescriptor | undefined,
  auth: AIProviderPlatformAuth | "",
  serverAllowed: boolean,
): AIProviderPlatformAuth | "" {
  if (descriptor?.auths.some((entry) => entry.auth === auth)) return auth;
  return defaultAuth(descriptor, serverAllowed);
}

/** The settings the platform and its way of signing in ask for, in the order they are shown. */
export function platformFields(
  descriptor: AIProviderPlatformDescriptor | undefined,
  auth: AIProviderPlatformAuth | "",
): AIProviderPlatformSetting[] {
  if (!descriptor) return [];
  const authEntry = descriptor.auths.find((entry) => entry.auth === auth);
  return [...descriptor.settings, ...(authEntry?.settings ?? [])];
}

/** Settings a platform asks for that need not be filled: a default stands in. */
const OPTIONAL_SETTINGS: Partial<Record<AIProviderPlatform, AIProviderPlatformSetting[]>> = {
  ClaudePlatformGoogleCloud: ["region"],
  Foundry: ["resource"],
};

export function settingRequired(
  platform: AIProviderPlatform,
  setting: AIProviderPlatformSetting,
  baseUrl: string,
): boolean {
  if (platform === "Foundry" && setting === "resource") {
    return baseUrl.trim() === "";
  }
  return !(OPTIONAL_SETTINGS[platform] ?? []).includes(setting);
}

/**
 * Whether a credential has to be stored. On a platform the way it signs in
 * says; directly, the kind does.
 */
export function credentialRequired(context: {
  kind: AIProviderKind;
  platform: AIProviderPlatform;
  auth: AIProviderPlatformAuth | "";
  descriptor: AIProviderPlatformDescriptor | undefined;
  kindRequiresKey: boolean;
}): boolean {
  if (effectivePlatform(context.kind, context.platform) === "Direct") {
    return context.kindRequiresKey;
  }
  const entry = context.descriptor?.auths.find((auth) => auth.auth === context.auth);
  return entry?.requiresCredential ?? true;
}

/**
 * The platform and settings a save or a test sends: only what the chosen
 * platform and way of signing in take, trimmed, so a value typed for another
 * platform is never sent to this one. A provider called directly sends none.
 */
export function platformPayload(context: {
  kind: AIProviderKind;
  platform: AIProviderPlatform;
  settings: PlatformSettingsDraft;
  descriptor: AIProviderPlatformDescriptor | undefined;
  serverAllowed: boolean;
}): { platform: AIProviderPlatform; platformSettings: AIProviderPlatformSettings | null } {
  const platform = effectivePlatform(context.kind, context.platform);
  if (platform === "Direct") {
    return { platform, platformSettings: null };
  }

  const auth = resolvedAuth(context.descriptor, context.settings.auth, context.serverAllowed);
  const fields = platformFields(context.descriptor, auth);
  const settings: AIProviderPlatformSettings = {
    auth: auth === "" ? "ApiKey" : auth,
    region: "",
    projectId: "",
    workspaceId: "",
    accessKeyId: "",
    resource: "",
    tenantId: "",
    clientId: "",
  };
  for (const field of fields) {
    settings[field] = context.settings[field].trim();
  }

  return { platform, platformSettings: settings };
}

/** A saved provider's settings as the editor holds them. */
export function platformSettingsDraft(
  saved: AIProviderPlatformSettings | null | undefined,
): PlatformSettingsDraft {
  if (!saved) return { ...EMPTY_PLATFORM_SETTINGS };
  return {
    auth: saved.auth,
    region: saved.region,
    projectId: saved.projectId,
    workspaceId: saved.workspaceId,
    accessKeyId: saved.accessKeyId,
    resource: saved.resource,
    tenantId: saved.tenantId,
    clientId: saved.clientId,
  };
}

/** A platform's settings in one line for a change review: the way it signs in, then each filled setting. */
export function platformSettingsSummary(
  settings: PlatformSettingsDraft | null | undefined,
): string {
  if (!settings) return "";
  const parts: string[] = settings.auth === "" ? [] : [PLATFORM_AUTH_LABELS[settings.auth]];
  for (const setting of SETTING_ORDER) {
    const value = settings[setting].trim();
    if (value !== "") parts.push(`${PLATFORM_SETTING_LABELS[setting]} ${value}`);
  }
  return parts.join(" · ");
}
