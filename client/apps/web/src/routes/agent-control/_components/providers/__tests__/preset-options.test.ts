import { describe, expect, it } from "vitest";
import type { AIProviderPreset } from "@/types/ai-provider";
import {
  CUSTOM_PRESET_VALUE,
  findPreset,
  groupPresets,
  presetDisplayName,
  presetHint,
  startingPreset,
} from "../preset-options";

/**
 * Fixtures follow the Go catalog (aiproviderhandler/catalog.go): a hosted
 * endpoint that needs a key, a self-hosted one that does not, and Bedrock,
 * which carries neither a base URL nor an example model.
 */
function preset(overrides: Partial<AIProviderPreset>): AIProviderPreset {
  return {
    key: "openai",
    label: "OpenAI",
    kind: "OpenAIResponses",
    baseUrl: "https://api.openai.com",
    structuredOutputMode: "JSONSchema",
    allowPrivateNetwork: false,
    requiresApiKey: true,
    selfHosted: false,
    exampleModel: "gpt-5-mini",
    notes: "",
    domain: "openai.com",
    ...overrides,
  } as AIProviderPreset;
}

const hosted = preset({});
const local = preset({
  key: "vllm",
  label: "vLLM (self-hosted)",
  kind: "OpenAIChat",
  baseUrl: "http://localhost:8000/v1",
  requiresApiKey: false,
  selfHosted: true,
  allowPrivateNetwork: true,
  exampleModel: "Qwen/Qwen3-32B",
  domain: "vllm.ai",
});
const bedrock = preset({
  key: "bedrock",
  label: "Amazon Bedrock",
  kind: "OpenAIChat",
  baseUrl: "",
  exampleModel: "",
  domain: "aws.amazon.com",
  notes: "Use the bedrock-mantle endpoint for your region.",
});

describe("preset options", () => {
  it("separates what you host from what you call", () => {
    const groups = groupPresets([hosted, local, bedrock]);

    expect(groups.map((group) => group.key)).toEqual(["hosted", "selfHosted"]);
    expect(groups[0].presets.map((entry) => entry.key)).toEqual(["openai", "bedrock"]);
    expect(groups[1].presets.map((entry) => entry.key)).toEqual(["vllm"]);
  });

  it("omits a group nothing belongs to", () => {
    expect(groupPresets([local]).map((group) => group.key)).toEqual(["selfHosted"]);
    expect(groupPresets([])).toEqual([]);
  });

  it("says what each deployment asks of you", () => {
    expect(presetHint(local)).toBe("local");
    expect(presetHint(hosted)).toBe("key");
    expect(presetHint(preset({ requiresApiKey: false }))).toBe("none");
  });

  it("drops the self-hosted suffix from the display name", () => {
    expect(presetDisplayName(local)).toBe("vLLM");
    expect(presetDisplayName(hosted)).toBe("OpenAI");
  });

  it("treats the empty key as the custom endpoint", () => {
    const all = [hosted, local];

    expect(findPreset(all, CUSTOM_PRESET_VALUE)).toBeNull();
    expect(findPreset(all, "vllm")).toBe(local);
    expect(findPreset(all, "nothing-like-this")).toBeNull();
  });
});

describe("startingPreset", () => {
  const openai = preset({ key: "openai" });
  const vllm = preset({ key: "vllm", label: "vLLM", selfHosted: true, requiresApiKey: false });

  it("opens on the preset a link names", () => {
    expect(startingPreset([openai, vllm], "vllm")).toBe(vllm);
  });

  it("opens on the first preset when the link names none, so New provider opens the panel", () => {
    expect(startingPreset([openai, vllm], null)).toBe(openai);
    expect(startingPreset([openai, vllm], "")).toBe(openai);
  });

  it("opens on the first preset when the link names one the catalog no longer offers", () => {
    expect(startingPreset([openai, vllm], "retired")).toBe(openai);
  });

  it("has nothing to open on while the catalog is empty", () => {
    expect(startingPreset([], null)).toBeNull();
  });
});
