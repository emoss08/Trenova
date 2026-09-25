import type { AccountingAppCredential } from "@/lib/graphql/accounting-sync";
import type { AccountingAppEnvironment } from "@trenova/graphql/generated/graphql";
import { z } from "zod";

export const ACCOUNTING_APP_ENVIRONMENTS = [
  "Sandbox",
  "Production",
] as const satisfies readonly AccountingAppEnvironment[];

const CLIENT_ID_PATTERN = /^[A-Za-z0-9._-]+$/;
const MAX_CLIENT_ID_LENGTH = 255;

type SavedApp = Pick<AccountingAppCredential, "clientId" | "environment"> | null;

export function accountingAppFormSchema(saved: SavedApp) {
  return z
    .object({
      environment: z.enum(ACCOUNTING_APP_ENVIRONMENTS, { error: "Choose an environment" }),
      clientId: z
        .string()
        .trim()
        .min(1, { error: "Client ID is required" })
        .max(MAX_CLIENT_ID_LENGTH, { error: "Client ID cannot be longer than 255 characters" })
        .regex(CLIENT_ID_PATTERN, {
          error: "Client ID can only contain letters, numbers, dots, dashes and underscores",
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
        error: "Enter the client secret that goes with this client ID",
        path: ["clientSecret"],
      },
    )
    .refine((value) => !(value.clearWebhookVerifierToken && value.webhookVerifierToken !== ""), {
      error: "Enter a verifier token or remove the saved one, not both",
      path: ["webhookVerifierToken"],
    });
}

export type AccountingAppFormValues = z.infer<ReturnType<typeof accountingAppFormSchema>>;

export function accountingAppFormDefaults(saved: SavedApp): AccountingAppFormValues {
  return {
    environment: saved?.environment ?? "Sandbox",
    clientId: saved?.clientId ?? "",
    clientSecret: "",
    webhookVerifierToken: "",
    clearWebhookVerifierToken: false,
  };
}
