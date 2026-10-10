import { describe, expect, it } from "vitest";
import type { AIProviderPreset } from "@/types/ai-provider";
import {
  cachePriceDefault,
  draftTestInput,
  editorBlocker,
  editorValuesFromPreset,
  providerEditorSchema,
  reportsCacheWrites,
  toSaveRequest,
  type ProviderEditorValues,
} from "../provider-editor-model";

const ANTHROPIC: AIProviderPreset = {
  key: "anthropic",
  label: "Anthropic",
  kind: "AnthropicMessages",
  baseUrl: "",
  structuredOutputMode: "JSONSchema",
  allowPrivateNetwork: false,
  requiresApiKey: true,
  selfHosted: false,
  exampleModel: "claude-sonnet-5",
  notes: "",
  domain: "anthropic.com",
  tasks: [],
  embeddingDimensions: 0,
};

const OLLAMA: AIProviderPreset = {
  ...ANTHROPIC,
  key: "ollama",
  label: "Ollama (self-hosted)",
  kind: "Ollama",
  baseUrl: "http://localhost:11434",
  allowPrivateNetwork: true,
  requiresApiKey: false,
  selfHosted: true,
  exampleModel: "qwen2.5:14b",
  domain: "ollama.com",
};

function values(overrides: Partial<ProviderEditorValues> = {}): ProviderEditorValues {
  return { ...editorValuesFromPreset(ANTHROPIC, []), ...overrides };
}

describe("editorValuesFromPreset", () => {
  it("remembers the preset it started from, so the panel shows it chosen", () => {
    expect(editorValuesFromPreset(ANTHROPIC, []).preset).toBe("anthropic");
  });

  it("joins the end of the chain, one step after the last provider", () => {
    expect(editorValuesFromPreset(ANTHROPIC, []).priority).toBe(10);
    expect(editorValuesFromPreset(ANTHROPIC, [{ priority: 30 }, { priority: 10 }]).priority).toBe(
      40,
    );
  });

  it("names a self-hosted preset as local and lets it reach the private network", () => {
    const local = editorValuesFromPreset(OLLAMA, []);
    expect(local.name).toBe("Local Ollama");
    expect(local.allowPrivateNetwork).toBe(true);
    expect(local.baseUrl).toBe("http://localhost:11434");
  });
});

describe("editorBlocker", () => {
  const create = { create: true, keyRequired: true, hasStoredKey: false };

  it("asks for a key on a new hosted provider, and not on an edit with one stored", () => {
    expect(editorBlocker(values(), create)).toBe("Paste an API key");
    expect(editorBlocker(values({ apiKey: "sk-ant-x" }), create)).toBeNull();
    expect(editorBlocker(values(), { ...create, create: false, hasStoredKey: true })).toBeNull();
  });

  it("asks for Private network before anything is saved to a private address", () => {
    expect(editorBlocker(values({ apiKey: "k", baseUrl: "http://10.0.0.5:8000" }), create)).toBe(
      "Fix the base URL",
    );
  });

  it("asks for a name first", () => {
    expect(editorBlocker(values({ name: "  " }), create)).toBe("Give it a name");
  });
});

describe("providerEditorSchema limits", () => {
  const paths = (input: ProviderEditorValues) => {
    const parsed = providerEditorSchema.safeParse(input);
    return parsed.success ? [] : parsed.error.issues.map((issue) => issue.path.join("."));
  };

  it("holds timeout and concurrency to the server's ranges", () => {
    expect(paths(values({ timeoutSeconds: "4" }))).toContain("timeoutSeconds");
    expect(paths(values({ timeoutSeconds: "601" }))).toContain("timeoutSeconds");
    expect(paths(values({ timeoutSeconds: "1.5" }))).toContain("timeoutSeconds");
    expect(paths(values({ maxConcurrent: "0" }))).toContain("maxConcurrent");
    expect(paths(values({ maxConcurrent: "65" }))).toContain("maxConcurrent");
    expect(paths(values({ timeoutSeconds: "90", maxConcurrent: "4" }))).toEqual([]);
  });

  it("takes no cap, or a cap above zero", () => {
    expect(paths(values({ monthlyCapUsd: "" }))).toEqual([]);
    expect(paths(values({ monthlyCapUsd: "200" }))).toEqual([]);
    expect(paths(values({ monthlyCapUsd: "0" }))).toContain("monthlyCapUsd");
    expect(paths(values({ monthlyCapUsd: "abc" }))).toContain("monthlyCapUsd");
  });
});

/**
 * The server contract: an absent apiKey keeps the stored key, keepPreviousKey only means
 * something when a key is being replaced, and a new provider starts off unless a test of
 * the draft passed.
 */
describe("toSaveRequest", () => {
  it("saves a new provider off unless its draft test passed", () => {
    expect(toSaveRequest(values({ apiKey: "k" }), { editing: false, enable: false }).enabled).toBe(
      false,
    );
    expect(toSaveRequest(values({ apiKey: "k" }), { editing: false, enable: true }).enabled).toBe(
      true,
    );
  });

  it("keeps the stored key on an edit with a blank key field", () => {
    const request = toSaveRequest(values({ apiKey: "", keepPreviousKey: true }), {
      editing: true,
      enable: false,
    });
    expect(request.apiKey).toBeUndefined();
    expect(request.keepPreviousKey).toBe(false);
  });

  it("keeps the old key only when one is being replaced and the switch is on", () => {
    expect(
      toSaveRequest(values({ apiKey: "new", keepPreviousKey: true }), {
        editing: true,
        enable: false,
      }).keepPreviousKey,
    ).toBe(true);
    expect(
      toSaveRequest(values({ apiKey: "new", keepPreviousKey: false }), {
        editing: true,
        enable: false,
      }).keepPreviousKey,
    ).toBe(false);
  });

  it("sends the limits as numbers and no cap as null", () => {
    const request = toSaveRequest(
      values({ apiKey: "k", timeoutSeconds: "90", maxConcurrent: "4", monthlyCapUsd: "" }),
      { editing: false, enable: false },
    );
    expect(request.timeoutSeconds).toBe(90);
    expect(request.maxConcurrent).toBe(4);
    expect(request.monthlyCapUsd).toBeNull();
    expect(
      toSaveRequest(values({ apiKey: "k", monthlyCapUsd: "200.5", onCap: "Stop" }), {
        editing: false,
        enable: false,
      }),
    ).toMatchObject({ monthlyCapUsd: 200.5, onCap: "Stop" });
  });
});

describe("draftTestInput", () => {
  it("leaves the key out so a saved provider's stored key is used", () => {
    const input = draftTestInput(values({ apiKey: "  " }), "aiprv_1");
    expect(input.endpoint).toMatchObject({ providerId: "aiprv_1", apiKey: null });
  });

  it("sends a typed key and the vector size only when it embeds", () => {
    const input = draftTestInput(
      values({ apiKey: "sk", tasks: ["AssistantChat"], embeddingDimensionsChoice: "1024" }),
      null,
    );
    expect(input.endpoint.apiKey).toBe("sk");
    expect(input.embeddingDimensions).toBeNull();
    expect(
      draftTestInput(values({ tasks: ["Embedding"], embeddingDimensionsChoice: "1024" }), null)
        .embeddingDimensions,
    ).toBe(1024);
  });
});

/**
 * An empty cache price charges a multiple of the input price on the server;
 * the field's placeholder shows that figure so nobody has to work it out.
 */
describe("cachePriceDefault", () => {
  it("reads a cached token at a tenth of input on every protocol", () => {
    expect(cachePriceDefault("AnthropicMessages", "3", "read")).toBe("0.3");
    expect(cachePriceDefault("OpenAIResponses", "1.25", "read")).toBe("0.125");
  });

  it("writes at one and a quarter times input for Anthropic, at input otherwise", () => {
    expect(cachePriceDefault("AnthropicMessages", "3", "write")).toBe("3.75");
    expect(cachePriceDefault("OpenAIChat", "3", "write")).toBe("3");
  });

  it("is blank until the input price is a number", () => {
    expect(cachePriceDefault("AnthropicMessages", "", "read")).toBe("");
    expect(cachePriceDefault("AnthropicMessages", "abc", "write")).toBe("");
  });

  it("knows only the Anthropic protocol reports cache writes", () => {
    expect(reportsCacheWrites("AnthropicMessages")).toBe(true);
    expect(reportsCacheWrites("OpenAIResponses")).toBe(false);
    expect(reportsCacheWrites("Ollama")).toBe(false);
  });
});
