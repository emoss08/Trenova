import { describe, expect, it } from "vitest";
import {
  accountingAppFormDefaults,
  accountingAppFormSchema,
  type AccountingAppFormValues,
} from "../accounting-app-schema";

const saved = { clientId: "ABsaved", environment: "Sandbox" as const };

function values(overrides: Partial<AccountingAppFormValues> = {}): AccountingAppFormValues {
  return {
    environment: "Sandbox",
    clientId: "ABsaved",
    clientSecret: "",
    webhookVerifierToken: "",
    clearWebhookVerifierToken: false,
    ...overrides,
  };
}

function issues(schema: ReturnType<typeof accountingAppFormSchema>, input: unknown) {
  const result = schema.safeParse(input);
  return result.success ? [] : result.error.issues.map((issue) => issue.path.join("."));
}

describe("accountingAppFormSchema", () => {
  it("needs a secret for an app that was never saved", () => {
    expect(issues(accountingAppFormSchema(null), values())).toEqual(["clientSecret"]);
    expect(issues(accountingAppFormSchema(null), values({ clientSecret: "s" }))).toEqual([]);
  });

  it("keeps the saved secret when the client ID and environment are unchanged", () => {
    expect(issues(accountingAppFormSchema(saved), values())).toEqual([]);
  });

  it("needs a new secret when the client ID or the environment changes", () => {
    const schema = accountingAppFormSchema(saved);
    expect(issues(schema, values({ clientId: "ABother" }))).toEqual(["clientSecret"]);
    expect(issues(schema, values({ environment: "Production" }))).toEqual(["clientSecret"]);
  });

  it("trims the keys before comparing and sending them", () => {
    const parsed = accountingAppFormSchema(saved).parse(
      values({ clientId: "  ABsaved ", clientSecret: "  s  ", webhookVerifierToken: " v " }),
    );
    expect(parsed.clientId).toBe("ABsaved");
    expect(parsed.clientSecret).toBe("s");
    expect(parsed.webhookVerifierToken).toBe("v");
  });

  it("treats a secret of spaces as no secret", () => {
    expect(issues(accountingAppFormSchema(null), values({ clientSecret: "   " }))).toEqual([
      "clientSecret",
    ]);
  });

  it("rejects client IDs Intuit could not have issued", () => {
    const schema = accountingAppFormSchema(null);
    expect(issues(schema, values({ clientId: "", clientSecret: "s" }))).toContain("clientId");
    expect(issues(schema, values({ clientId: "has space", clientSecret: "s" }))).toContain(
      "clientId",
    );
    expect(issues(schema, values({ clientId: "A".repeat(256), clientSecret: "s" }))).toContain(
      "clientId",
    );
    expect(issues(schema, values({ clientId: "AB.x-y_1", clientSecret: "s" }))).toEqual([]);
  });

  it("rejects an environment the server does not know", () => {
    expect(
      issues(accountingAppFormSchema(null), {
        ...values({ clientSecret: "s" }),
        environment: "Staging",
      }),
    ).toContain("environment");
  });

  it("will not both set and remove the verifier token", () => {
    expect(
      issues(
        accountingAppFormSchema(saved),
        values({ webhookVerifierToken: "v", clearWebhookVerifierToken: true }),
      ),
    ).toEqual(["webhookVerifierToken"]);
    expect(
      issues(accountingAppFormSchema(saved), values({ clearWebhookVerifierToken: true })),
    ).toEqual([]);
  });
});

describe("accountingAppFormDefaults", () => {
  it("starts a new app on sandbox with nothing filled in", () => {
    expect(accountingAppFormDefaults(null)).toEqual(values({ clientId: "" }));
  });

  it("starts an existing app from what was saved, never the secrets", () => {
    expect(accountingAppFormDefaults({ clientId: "ABprod", environment: "Production" })).toEqual(
      values({ clientId: "ABprod", environment: "Production" }),
    );
  });
});
