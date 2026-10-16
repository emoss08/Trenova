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

const secondsSchema = z.number().int().nonnegative().catch(0).default(0);

export const SESSION_IDLE_POLICY_OFF = {
  idleTimeoutSeconds: 0,
  idleWarningSeconds: 0,
  keepAliveSeconds: 0,
} as const satisfies Record<string, number>;

/**
 * How long a signed-in browser may sit untouched before it signs the person out, how long
 * before that it warns them, and how often it tells the server they are still there. The
 * server ends an idle session on its own as well; these only let the browser end it first,
 * with a warning, instead of leaving the next request to find it gone. Any value that
 * cannot describe a working timer turns the browser's half off rather than guessing.
 */
export const sessionIdlePolicySchema = z
  .object({
    idleTimeoutSeconds: secondsSchema,
    idleWarningSeconds: secondsSchema,
    keepAliveSeconds: secondsSchema,
  })
  .transform((policy) =>
    policy.idleTimeoutSeconds > 0 &&
    policy.idleWarningSeconds > 0 &&
    policy.idleWarningSeconds < policy.idleTimeoutSeconds &&
    policy.keepAliveSeconds > 0
      ? policy
      : { ...SESSION_IDLE_POLICY_OFF },
  );

export type SessionIdlePolicy = z.infer<typeof sessionIdlePolicySchema>;

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
  session: sessionIdlePolicySchema
    .nullish()
    .catch(null)
    .transform((value) => value ?? { ...SESSION_IDLE_POLICY_OFF }),
});

export type PublicConfig = z.infer<typeof publicConfigSchema>;

export const SELF_HOSTED_PUBLIC_CONFIG: Readonly<PublicConfig> = {
  platformMode: "self_hosted",
  signupEnabled: false,
  turnstileSiteKey: "",
  termsUrl: "",
  privacyUrl: "",
  freePlan: { limits: {} },
  session: { ...SESSION_IDLE_POLICY_OFF },
};

export function isCloudPlatform(config: Pick<PublicConfig, "platformMode"> | undefined): boolean {
  return config?.platformMode === "cloud";
}

export function isSignupAvailable(
  config: Pick<PublicConfig, "platformMode" | "signupEnabled"> | undefined,
): boolean {
  return isCloudPlatform(config) && config?.signupEnabled === true;
}
