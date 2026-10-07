import { describe, expect, it } from "vitest";
import type { AIProviderPreset } from "@/types/ai-provider";
import { presetValues, providerFormDefaults } from "../provider-form-schema";

const base = {
  key: "ollama",
  label: "Ollama (self-hosted)",
  kind: "Ollama",
  baseUrl: "http://localhost:11434",
  structuredOutputMode: "JSONSchema",
  allowPrivateNetwork: true,
  exampleModel: "qwen3:32b",
  tasks: [],
  embeddingDimensions: 0,
  embeddingInputStyle: "None",
} as unknown as AIProviderPreset;

describe("presetValues", () => {
  it("fills the endpoint, model and a name from the preset", () => {
    const next = presetValues(base, providerFormDefaults);

    expect(next).toMatchObject({
      preset: "ollama",
      kind: "Ollama",
      baseUrl: "http://localhost:11434",
      allowPrivateNetwork: true,
      model: "qwen3:32b",
      name: "Ollama",
    });
  });

  it("keeps a name the person typed and takes an embedding task away for a text preset", () => {
    const next = presetValues(base, {
      ...providerFormDefaults,
      name: "Workstation",
      tasks: ["Embedding"],
      embeddingDimensionsChoice: "768",
    });

    expect(next.name).toBe("Workstation");
    expect(next.tasks).toBeNull();
    expect(next.embeddingDimensionsChoice).toBe("");
  });

  it("brings an embedding preset's task and vector size", () => {
    const next = presetValues(
      { ...base, tasks: ["Embedding"], embeddingDimensions: 768 } as unknown as AIProviderPreset,
      providerFormDefaults,
    );

    expect(next.tasks).toEqual(["Embedding"]);
    expect(next.embeddingDimensionsChoice).toBe("768");
  });
});
