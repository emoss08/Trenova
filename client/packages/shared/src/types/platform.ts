import { z } from "zod";
import { translate } from "@trenova/shared/i18n/runtime";

export const PLATFORM_MODES = [
  "self_hosted",
  "community",
  "enterprise",
  "development",
  "cloud",
] as const;

export type PlatformMode = (typeof PLATFORM_MODES)[number];

const platformModeSchema = z.enum(PLATFORM_MODES).catch("self_hosted");

const limitValueSchema = z.union([z.number(), z.string()]).transform((value, ctx) => {
  const parsed = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(parsed)) {
    ctx.addIssue({ code: "custom", message: translate("Limit must be a number") });
    return z.NEVER;
  }
  return parsed;
});

export const freePlanSummarySchema = z.object({
  limits: z.record(z.string(), limitValueSchema).catch({}).default({}),
});

const urlOrEmptySchema = z
  .string()
  .nullish()
  .transform((value) => value?.trim() ?? "")
  .pipe(z.union([z.url({ protocol: /^https?$/ }), z.literal("")]))
  .catch("");

export const publicConfigSchema = z.object({
  platformMode: platformModeSchema.default("self_hosted"),
  signupEnabled: z.boolean().catch(false).default(false),
  turnstileSiteKey: z
    .string()
    .nullish()
    .transform((value) => value?.trim() ?? "")
    .catch(""),
  termsUrl: urlOrEmptySchema,
  privacyUrl: urlOrEmptySchema,
  freePlan: freePlanSummarySchema.nullish().transform((value) => value ?? { limits: {} }),
});

export type PublicConfig = z.infer<typeof publicConfigSchema>;

export const SELF_HOSTED_PUBLIC_CONFIG: Readonly<PublicConfig> = {
  platformMode: "self_hosted",
  signupEnabled: false,
  turnstileSiteKey: "",
  termsUrl: "",
  privacyUrl: "",
  freePlan: { limits: {} },
};

export function isCloudPlatform(config: Pick<PublicConfig, "platformMode"> | undefined): boolean {
  return config?.platformMode === "cloud";
}

export function isSignupAvailable(
  config: Pick<PublicConfig, "platformMode" | "signupEnabled"> | undefined,
): boolean {
  return isCloudPlatform(config) && config?.signupEnabled === true;
}
