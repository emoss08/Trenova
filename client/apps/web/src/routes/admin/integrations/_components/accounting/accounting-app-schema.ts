import type { AccountingAppCredential } from "@/lib/graphql/accounting-sync";
import type { AccountingAppEnvironment } from "@trenova/graphql/generated/graphql";
import { z } from "zod";
import { translate } from "@trenova/shared/i18n/runtime";

export const ACCOUNTING_APP_ENVIRONMENTS = [
  "Sandbox",
  "Production",
] as const satisfies readonly AccountingAppEnvironment[];

const CLIENT_ID_PATTERN = /^[A-Za-z0-9._-]+$/;
const MAX_CLIENT_ID_LENGTH = 255;

type SavedApp = Pick<AccountingAppCredential, "clientId" | "environment"> | null;

export function defaultAccountingAppEnvironment(
  environments: readonly AccountingAppEnvironment[],
): AccountingAppEnvironment {
  return environments[0] ?? "Production";
}

export function accountingAppFormSchema(
  saved: SavedApp,
  environments: readonly AccountingAppEnvironment[] = ACCOUNTING_APP_ENVIRONMENTS,
) {
  return z
    .object({
      environment: z
        .enum(ACCOUNTING_APP_ENVIRONMENTS, { error: () => translate("Choose an environment") })
        .refine((value) => environments.includes(value), {
          error: () => translate("Choose an environment"),
        }),
      clientId: z
        .string()
        .trim()
        .min(1, { error: () => translate("Client ID is required") })
        .max(MAX_CLIENT_ID_LENGTH, {
          error: () => translate("Client ID cannot be longer than 255 characters"),
        })
        .regex(CLIENT_ID_PATTERN, {
          error: () =>
            translate("Client ID can only contain letters, numbers, dots, dashes and underscores"),
        }),
      clientSecret: z.string().trim(),
      webhookVerifierToken: z.string().trim(),
      clearWebhookVerifierToken: z.boolean(),
    })
    .refine(
      (value) =>
        value.clientSecret !== "" ||
        (saved !== null &&
          saved.clientId === value.clientId &&
          saved.environment === value.environment),
      {
        error: () => translate("Enter the client secret that goes with this client ID"),
        path: ["clientSecret"],
      },
    )
    .refine((value) => !(value.clearWebhookVerifierToken && value.webhookVerifierToken !== ""), {
      error: () => translate("Enter a new key or remove the saved one, not both"),
      path: ["webhookVerifierToken"],
    });
}

export type AccountingAppFormValues = z.infer<ReturnType<typeof accountingAppFormSchema>>;

export function accountingAppFormDefaults(
  saved: SavedApp,
  environments: readonly AccountingAppEnvironment[] = ACCOUNTING_APP_ENVIRONMENTS,
): AccountingAppFormValues {
  return {
    environment:
      saved && environments.includes(saved.environment)
        ? saved.environment
        : defaultAccountingAppEnvironment(environments),
    clientId: saved?.clientId ?? "",
    clientSecret: "",
    webhookVerifierToken: "",
    clearWebhookVerifierToken: false,
  };
}
