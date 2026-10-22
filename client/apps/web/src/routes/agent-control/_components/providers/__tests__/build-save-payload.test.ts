import { describe, expect, it } from "vitest";
import {
  buildSavePayload,
  type ProviderFormValues,
} from "@/routes/agent-control/_components/providers/build-save-payload";

function formValues(overrides: Partial<ProviderFormValues> = {}): ProviderFormValues {
  return {
    preset: "",
    name: "Local qwen",
    description: "",
    kind: "OpenAIChat",
    platform: "Direct",
    platformSettings: null,
    baseUrl: "http://10.0.0.5:8000/v1",
    model: "qwen3:32b",
    apiKey: "",
    allowPrivateNetwork: true,
    structuredOutputMode: "Prompted",
    reasoningEffort: "Off",
    thinkingStyle: "Auto",
    advisorEnabled: false,
    advisorModel: "",
    scripts: false,
    scriptWrites: false,
    extraBodyText: "",
    inputCostPerMillion: null,
    outputCostPerMillion: null,
    cacheReadCostPerMillion: null,
    cacheWriteCostPerMillion: null,
    maxTokens: 8192,
    tasks: null,
    priority: 100,
    embeddingDimensionsChoice: "",
    embeddingInputStyle: "None",
    trusted: false,
    enabled: true,
    timeoutSeconds: 60,
    maxConcurrent: 8,
    monthlyCapUsd: null,
    onCap: "Next",
    keepPreviousKey: false,
    version: 0,
    ...overrides,
  };
}

/**
 * The server contract (services/tms/internal/modules/agent/ports/agentports/aiprovider_services.go):
 * an absent apiKey leaves the stored credential alone, an empty string clears it.
 * The edit form never receives the existing secret, so these cases decide whether
 * a credential survives a routine edit.
 */
describe("buildSavePayload credential handling", () => {
  it("omits the key entirely when editing with a blank field", () => {
    const payload = buildSavePayload(formValues({ apiKey: "" }), true);

    expect(payload.apiKey).toBeUndefined();
    expect("apiKey" in payload && payload.apiKey === "").toBe(false);
  });

  it("omits the key when editing and the field holds only whitespace", () => {
    const payload = buildSavePayload(formValues({ apiKey: "   " }), true);

    expect(payload.apiKey).toBeUndefined();
  });

  it("sends an empty key when creating with a blank field", () => {
    // A self-hosted endpoint commonly needs no credential, and there is nothing
    // stored to preserve on a create.
    const payload = buildSavePayload(formValues({ apiKey: "" }), false);

    expect(payload.apiKey).toBe("");
  });

  it("sends the key when editing and a new one is supplied", () => {
    const payload = buildSavePayload(formValues({ apiKey: "sk-new-secret" }), true);

    expect(payload.apiKey).toBe("sk-new-secret");
  });

  it("sends the key when creating and one is supplied", () => {
    const payload = buildSavePayload(formValues({ apiKey: "sk-new-secret" }), false);

    expect(payload.apiKey).toBe("sk-new-secret");
  });

  it("trims a key before sending it", () => {
    const payload = buildSavePayload(formValues({ apiKey: "  sk-padded  " }), false);

    expect(payload.apiKey).toBe("sk-padded");
  });
});

describe("buildSavePayload field mapping", () => {
  it("normalizes an empty task selection to an array", () => {
    // The checkbox group stores null for an empty selection; the server expects
    // a list.
    const payload = buildSavePayload(formValues({ tasks: null }), false);

    expect(payload.tasks).toEqual([]);
  });

  it("preserves a task selection", () => {
    const payload = buildSavePayload(
      formValues({ tasks: ["DocumentClassification", "General"] }),
      false,
    );

    expect(payload.tasks).toEqual(["DocumentClassification", "General"]);
  });

  it("drops the transient preset field", () => {
    const payload = buildSavePayload(formValues({ preset: "vllm" }), false);

    expect(payload).not.toHaveProperty("preset");
  });

  it("carries the remaining configuration through unchanged", () => {
    const payload = buildSavePayload(
      formValues({
        name: "Hosted Claude",
        kind: "AnthropicMessages",
        baseUrl: "https://api.anthropic.com",
        model: "claude-opus-5",
        allowPrivateNetwork: false,
        structuredOutputMode: "JSONSchema",
        maxTokens: 4096,
        priority: 20,
        trusted: true,
        enabled: true,
        version: 7,
      }),
      true,
    );

    expect(payload).toMatchObject({
      name: "Hosted Claude",
      kind: "AnthropicMessages",
      baseUrl: "https://api.anthropic.com",
      model: "claude-opus-5",
      allowPrivateNetwork: false,
      structuredOutputMode: "JSONSchema",
      maxTokens: 4096,
      priority: 20,
      trusted: true,
      enabled: true,
      version: 7,
    });
  });
});

describe("extra request fields", () => {
  it("sends the vendor fields the endpoint needs, parsed from what was typed", () => {
    // The body from NVIDIA's own example for a Nemotron model: thinking is
    // switched on through the chat template and the reasoning budget is its
    // own field, neither of which the protocol defines.
    const payload = buildSavePayload(
      formValues({
        extraBodyText: '{"chat_template_kwargs":{"enable_thinking":true},"reasoning_budget":16384}',
      }),
      false,
    );

    expect(payload.extraBody).toEqual({
      chat_template_kwargs: { enable_thinking: true },
      reasoning_budget: 16384,
    });
  });

  it("sends nothing rather than an empty object when the box is blank", () => {
    expect(buildSavePayload(formValues({ extraBodyText: "" }), false).extraBody).toBeNull();
    expect(buildSavePayload(formValues({ extraBodyText: "   " }), false).extraBody).toBeNull();
    expect(buildSavePayload(formValues({ extraBodyText: "{}" }), false).extraBody).toBeNull();
  });

  it("sends nothing for a value the form's validation would have refused", () => {
    // An array or a bare number is not a set of request fields. The schema
    // refuses both before a save; if one reaches here it means validation
    // was bypassed, and no vendor fields is the safe reading.
    expect(buildSavePayload(formValues({ extraBodyText: "[1,2]" }), false).extraBody).toBeNull();
    expect(buildSavePayload(formValues({ extraBodyText: "7" }), false).extraBody).toBeNull();
    expect(buildSavePayload(formValues({ extraBodyText: "{oops" }), false).extraBody).toBeNull();
  });
});

/**
 * The server refuses any thinking style but Auto on a protocol other than
 * Anthropic Messages (aiprovider.Provider.Validate), and the field is shown only
 * for that protocol. A style picked before the protocol changed must not ride
 * along unseen and fail the save.
 */
describe("buildSavePayload thinking style", () => {
  it("keeps the style chosen for an Anthropic provider", () => {
    const payload = buildSavePayload(
      formValues({ kind: "AnthropicMessages", thinkingStyle: "Effort" }),
      false,
    );

    expect(payload.thinkingStyle).toBe("Effort");
  });

  it.each(["OpenAIChat", "OpenAIResponses"] as const)(
    "sends Auto for a %s provider whatever the field holds",
    (kind) => {
      const payload = buildSavePayload(formValues({ kind, thinkingStyle: "Budget" }), false);

      expect(payload.thinkingStyle).toBe("Auto");
    },
  );
});

/**
 * Only the Anthropic protocol reports prompt tokens written to its cache, so a
 * cache write price is kept for it and dropped for any other, the way the
 * thinking style is. A cache read price applies to every protocol.
 */
describe("buildSavePayload cache prices", () => {
  it("keeps both cache prices for Anthropic Messages", () => {
    const payload = buildSavePayload(
      formValues({
        kind: "AnthropicMessages",
        cacheReadCostPerMillion: 0.3,
        cacheWriteCostPerMillion: 3.75,
      }),
      true,
    );

    expect(payload.cacheReadCostPerMillion).toBe(0.3);
    expect(payload.cacheWriteCostPerMillion).toBe(3.75);
  });

  it("drops a cache write price for a protocol that reports no writes", () => {
    const payload = buildSavePayload(
      formValues({
        kind: "OpenAIResponses",
        cacheReadCostPerMillion: 0.2,
        cacheWriteCostPerMillion: 3,
      }),
      true,
    );

    expect(payload.cacheReadCostPerMillion).toBe(0.2);
    expect(payload.cacheWriteCostPerMillion).toBeNull();
  });
});

/**
 * The server contract (aiprovider.Provider validation): the advisor is taken
 * only by an Anthropic Messages provider on the Claude API or Claude Platform
 * on AWS that handles AssistantChat; scripts only by a provider that handles
 * AssistantChat; script writes only with scripts. A setting the provider
 * cannot take is saved off rather than refused.
 */
describe("buildSavePayload advisor and scripts", () => {
  const claudeChat = {
    kind: "AnthropicMessages",
    platform: "Direct",
    tasks: ["AssistantChat"],
  } satisfies Partial<ProviderFormValues>;

  it("sends all four settings for a Claude API chat provider", () => {
    const payload = buildSavePayload(
      formValues({
        ...claudeChat,
        advisorEnabled: true,
        advisorModel: "  claude-opus-5-5  ",
        scripts: true,
        scriptWrites: true,
      }),
      true,
    );

    expect({
      advisorEnabled: payload.advisorEnabled,
      advisorModel: payload.advisorModel,
      scripts: payload.scripts,
      scriptWrites: payload.scriptWrites,
    }).toEqual({
      advisorEnabled: true,
      advisorModel: "claude-opus-5-5",
      scripts: true,
      scriptWrites: true,
    });
  });

  it("keeps the advisor on Claude Platform on AWS", () => {
    const payload = buildSavePayload(
      formValues({
        ...claudeChat,
        platform: "ClaudePlatformAWS",
        platformSettings: {
          auth: "AccessKey",
          region: "us-east-1",
          projectId: "",
          workspaceId: "wrkspc_1",
          accessKeyId: "AKIA",
          resource: "",
          tenantId: "",
          clientId: "",
        },
        advisorEnabled: true,
      }),
      true,
    );

    expect(payload.advisorEnabled).toBe(true);
  });

  it.each(["Bedrock", "VertexAI", "ClaudePlatformGoogleCloud", "Foundry"] as const)(
    "turns the advisor off on %s, which does not serve it",
    (platform) => {
      const payload = buildSavePayload(
        formValues({ ...claudeChat, platform, advisorEnabled: true, advisorModel: "claude-x" }),
        true,
      );

      expect(payload.advisorEnabled).toBe(false);
      expect(payload.advisorModel).toBe("claude-x");
    },
  );

  it.each(["OpenAIChat", "OpenAIResponses"] as const)(
    "sends no advisor for %s, which is not Anthropic's protocol",
    (kind) => {
      const payload = buildSavePayload(
        formValues({
          kind,
          tasks: ["AssistantChat"],
          advisorEnabled: true,
          advisorModel: "claude-opus-5-5",
        }),
        true,
      );

      expect(payload.advisorEnabled).toBe(false);
      expect(payload.advisorModel).toBe("");
    },
  );

  it("turns the advisor and scripts off for a provider that does not handle chat", () => {
    const payload = buildSavePayload(
      formValues({
        ...claudeChat,
        tasks: ["DocumentExtraction"],
        advisorEnabled: true,
        scripts: true,
        scriptWrites: true,
      }),
      true,
    );

    expect(payload.advisorEnabled).toBe(false);
    expect(payload.scripts).toBe(false);
    expect(payload.scriptWrites).toBe(false);
  });

  it("sends script writes off when scripts are off", () => {
    const payload = buildSavePayload(
      formValues({ ...claudeChat, scripts: false, scriptWrites: true }),
      true,
    );

    expect(payload.scripts).toBe(false);
    expect(payload.scriptWrites).toBe(false);
  });

  it("keeps scripts for a chat provider of any protocol", () => {
    const payload = buildSavePayload(
      formValues({ kind: "OpenAIChat", tasks: ["AssistantChat"], scripts: true }),
      false,
    );

    expect(payload.scripts).toBe(true);
    expect(payload.scriptWrites).toBe(false);
  });
});
