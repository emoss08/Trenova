import { describe, expect, it } from "vitest";
import {
  buildSavePayload,
  type ProviderFormValues,
} from "@/routes/agent-control/_components/providers/build-save-payload";

function formValues(overrides: Partial<ProviderFormValues> = {}): ProviderFormValues {
  return {
    preset: "",
    name: "Local Qwen",
    description: "",
    kind: "OpenAIChat",
    baseUrl: "http://10.0.0.5:8000/v1",
    model: "qwen3:32b",
    apiKey: "",
    allowPrivateNetwork: true,
    structuredOutputMode: "Prompted",
    reasoningEffort: "Off",
    inputCostPerMillion: null,
    outputCostPerMillion: null,
    maxTokens: 8192,
    tasks: null,
    priority: 100,
    trusted: false,
    enabled: true,
    version: 0,
    ...overrides,
  };
}

/**
 * The server contract (services/tms/internal/core/ports/services/aiprovider.go):
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
