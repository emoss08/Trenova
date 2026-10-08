import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import { parseEmbeddingDimensions } from "@/routes/agent-control/_components/providers/build-save-payload";
import {
  editorValuesFromPreset,
  providerEditorSchema,
  toSaveRequest,
  type ProviderEditorValues,
} from "@/routes/agent-control/_components/providers/provider-editor-model";
import {
  embeddingIssues,
  kindSupportsEmbedding,
} from "@/routes/agent-control/_components/providers/provider-form-schema";
import { goEnumValues, repoRoot } from "@/test/go-source";
import {
  type AIProviderPreset,
  EMBEDDING_DIMENSIONS,
  embeddingInputStyleSchema,
  saveAIProviderRequestSchema,
} from "@/types/ai-provider";

const ENUMS_FILE = "services/tms/internal/core/domain/aiprovider/enums.go";

const VOYAGE: AIProviderPreset = {
  key: "voyage",
  label: "Voyage AI",
  kind: "OpenAIChat",
  baseUrl: "https://api.voyageai.com/v1",
  structuredOutputMode: "JSONSchema",
  allowPrivateNetwork: false,
  requiresApiKey: true,
  selfHosted: false,
  exampleModel: "voyage-3.5",
  notes: "",
  domain: "voyageai.com",
  tasks: ["Embedding"],
  embeddingDimensions: 1024,
  embeddingInputStyle: "VoyageInputType",
};

function embeddingForm(overrides: Partial<ProviderEditorValues> = {}): ProviderEditorValues {
  return {
    ...editorValuesFromPreset(VOYAGE, []),
    name: "Voyage",
    apiKey: "pa-test",
    ...overrides,
  };
}

function issuePaths(values: ProviderEditorValues): string[] {
  const parsed = providerEditorSchema.safeParse(values);
  if (parsed.success) {
    return [];
  }

  return parsed.error.issues.map((issue) => issue.path.join("."));
}

/**
 * The dimension list and the input styles are server-side sets. A copy here
 * that falls behind refuses a provider the server saved, so both are read
 * from the Go source rather than listed again by hand.
 */
describe("embedding sets mirror the server", () => {
  it("offers every input style the server accepts, and no other", () => {
    const serverStyles = goEnumValues({ file: ENUMS_FILE, typeName: "EmbeddingInputStyle" });

    expect([...embeddingInputStyleSchema.options].sort()).toEqual([...serverStyles].sort());
  });

  it("allows exactly the dimensions the server allows", () => {
    const source = readFileSync(join(repoRoot(), ENUMS_FILE), "utf8");
    const match = /allowedEmbeddingDimensions = \[\.\.\.\]int\{([^}]*)\}/.exec(source);
    expect(match).not.toBeNull();

    const serverDimensions = (match?.[1] ?? "")
      .split(",")
      .map((value) => Number(value.trim()))
      .filter((value) => Number.isInteger(value) && value > 0);

    expect([...EMBEDDING_DIMENSIONS]).toEqual(serverDimensions);
  });
});

describe("embeddingIssues", () => {
  it("has nothing to say about a provider that does not embed", () => {
    expect(
      embeddingIssues({
        kind: "AnthropicMessages",
        tasks: ["AssistantChat"],
        embeddingDimensionsChoice: "",
      }),
    ).toEqual([]);
  });

  it("accepts a complete embedding provider", () => {
    expect(
      embeddingIssues({ kind: "Ollama", tasks: ["Embedding"], embeddingDimensionsChoice: "768" }),
    ).toEqual([]);
  });

  it("refuses the Embedding task on Anthropic", () => {
    const issues = embeddingIssues({
      kind: "AnthropicMessages",
      tasks: ["Embedding"],
      embeddingDimensionsChoice: "1024",
    });

    expect(issues.map((issue) => issue.path)).toEqual(["tasks"]);
  });

  it("refuses an embedding model that also writes text", () => {
    const issues = embeddingIssues({
      kind: "OpenAIChat",
      tasks: ["Embedding", "General"],
      embeddingDimensionsChoice: "1024",
    });

    expect(issues.map((issue) => issue.path)).toEqual(["tasks"]);
  });

  it("requires the vector size", () => {
    const issues = embeddingIssues({
      kind: "OpenAIChat",
      tasks: ["Embedding"],
      embeddingDimensionsChoice: "",
    });

    expect(issues.map((issue) => issue.path)).toEqual(["embeddingDimensionsChoice"]);
  });

  it("knows which protocols have an embedding endpoint", () => {
    expect(kindSupportsEmbedding("AnthropicMessages")).toBe(false);
    expect(kindSupportsEmbedding("OpenAIResponses")).toBe(true);
    expect(kindSupportsEmbedding("OpenAIChat")).toBe(true);
    expect(kindSupportsEmbedding("Ollama")).toBe(true);
  });
});

describe("providerEditorSchema with embeddings", () => {
  it("accepts a Voyage embedding provider", () => {
    expect(issuePaths(embeddingForm())).toEqual([]);
  });

  it("refuses a size the indexes are not built for", () => {
    expect(issuePaths(embeddingForm({ embeddingDimensionsChoice: "3072" }))).toContain(
      "embeddingDimensionsChoice",
    );
  });

  it("refuses an embedding provider with no size", () => {
    expect(issuePaths(embeddingForm({ embeddingDimensionsChoice: "" }))).toContain(
      "embeddingDimensionsChoice",
    );
  });

  it("refuses Embedding on the Anthropic protocol", () => {
    expect(issuePaths(embeddingForm({ kind: "AnthropicMessages" }))).toContain("tasks");
  });

  it("leaves a text provider alone", () => {
    expect(
      issuePaths(
        embeddingForm({
          tasks: ["AssistantChat"],
          embeddingDimensionsChoice: "",
          embeddingInputStyle: "None",
        }),
      ),
    ).toEqual([]);
  });
});

describe("toSaveRequest with embeddings", () => {
  it("sends the size as a number with the input style", () => {
    const payload = toSaveRequest(embeddingForm(), { editing: false, enable: true });

    expect(payload.tasks).toEqual(["Embedding"]);
    expect(payload.embeddingDimensions).toBe(1024);
    expect(payload.embeddingInputStyle).toBe("VoyageInputType");
    expect(saveAIProviderRequestSchema.safeParse(payload).success).toBe(true);
  });

  it("sends neither for a provider that does not embed", () => {
    const payload = toSaveRequest(
      embeddingForm({ tasks: ["General"], embeddingDimensionsChoice: "1024" }),
      { editing: false, enable: true },
    );

    expect(payload.embeddingDimensions).toBeNull();
    expect(payload.embeddingInputStyle).toBe("None");
  });

  it("reads the size choice", () => {
    expect(parseEmbeddingDimensions("")).toBeNull();
    expect(parseEmbeddingDimensions("  ")).toBeNull();
    expect(parseEmbeddingDimensions("768")).toBe(768);
    expect(parseEmbeddingDimensions("12.5")).toBeNull();
  });
});
