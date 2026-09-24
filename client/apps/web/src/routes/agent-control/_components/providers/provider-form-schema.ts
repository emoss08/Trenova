import {
  aiTaskSchema,
  EMBEDDING_DIMENSIONS,
  saveAIProviderRequestSchema,
  type AIProviderKind,
} from "@/types/ai-provider";
import type { AIProviderRow } from "@/lib/graphql/ai-provider";
import { z } from "zod";
import type { ProviderFormValues } from "./build-save-payload";

/**
 * The save contract plus the two things only the form knows: which preset
 * was picked (never sent) and a nullable task list, which is how the checkbox
 * group represents "none selected".
 */
export const providerFormSchema = saveAIProviderRequestSchema
  .omit({ tasks: true, extraBody: true, embeddingDimensions: true })
  .extend({
    embeddingDimensionsChoice: z
      .string()
      .default("")
      .refine(
        (value) =>
          value === "" || (EMBEDDING_DIMENSIONS as readonly number[]).includes(Number(value)),
        { message: "Embedding dimensions must be 768, 1024 or 1536" },
      ),
    preset: z.string().default(""),
    tasks: z.array(aiTaskSchema).nullable().default(null),
    /**
     * The vendor fields are edited as JSON text rather than as an object,
     * because that is how every provider's own documentation writes them —
     * the NVIDIA example is a JSON body to paste. Blank means none; anything
     * else must parse to an object, since a bare array or number is not a set
     * of request fields.
     */
    extraBodyText: z
      .string()
      .default("")
      .superRefine((value, ctx) => {
        const trimmed = value.trim();
        if (trimmed === "") {
          return;
        }
        let parsed: unknown;
        try {
          parsed = JSON.parse(trimmed);
        } catch {
          ctx.addIssue({ code: "custom", message: "This is not valid JSON" });
          return;
        }
        if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
          ctx.addIssue({
            code: "custom",
            message: 'Extra fields must be a JSON object, like {"top_k": 40}',
          });
        }
      }),
  })
  .superRefine((values, ctx) => {
    for (const issue of embeddingIssues(values)) {
      ctx.addIssue({ code: "custom", path: [issue.path], message: issue.message });
    }
  });

/** Protocols with no embedding endpoint. Anthropic's Messages API has none. */
const KINDS_WITHOUT_EMBEDDINGS: readonly AIProviderKind[] = ["AnthropicMessages"];

export function kindSupportsEmbedding(kind: AIProviderKind): boolean {
  return !KINDS_WITHOUT_EMBEDDINGS.includes(kind);
}

type EmbeddingIssue = {
  path: "tasks" | "embeddingDimensionsChoice";
  message: string;
};

/**
 * The rules the server applies to an embedding provider, checked while the
 * form is open: the protocol must have an embedding endpoint, an embedding
 * model serves nothing else, and its vector size must be chosen because the
 * index stores vectors of one size.
 */
export function embeddingIssues(values: {
  kind: AIProviderKind;
  tasks: readonly string[] | null;
  embeddingDimensionsChoice: string;
}): EmbeddingIssue[] {
  const tasks = values.tasks ?? [];
  if (!tasks.includes("Embedding")) {
    return [];
  }

  const issues: EmbeddingIssue[] = [];
  if (!kindSupportsEmbedding(values.kind)) {
    issues.push({
      path: "tasks",
      message: "This protocol has no embedding endpoint, so it cannot serve the Embedding task",
    });
  }
  if (tasks.length > 1) {
    issues.push({
      path: "tasks",
      message:
        "An embedding model cannot also serve tasks that write text; add a separate provider for those",
    });
  }
  if (values.embeddingDimensionsChoice.trim() === "") {
    issues.push({
      path: "embeddingDimensionsChoice",
      message: "Embedding dimensions are required for a provider that serves the Embedding task",
    });
  }

  return issues;
}

export const providerFormDefaults: ProviderFormValues = {
  preset: "",
  name: "",
  description: "",
  kind: "OpenAIChat",
  baseUrl: "",
  model: "",
  apiKey: "",
  allowPrivateNetwork: false,
  structuredOutputMode: "JSONSchema",
  reasoningEffort: "Off",
  extraBodyText: "",
  inputCostPerMillion: null,
  outputCostPerMillion: null,
  maxTokens: 8192,
  tasks: null,
  priority: 100,
  embeddingDimensionsChoice: "",
  embeddingInputStyle: "None",
  trusted: false,
  enabled: true,
  version: 0,
};

/** What the edit panel loads into the form: the row's values plus what the panel header reads. */
export type ProviderPanelRow = ProviderFormValues & {
  id: string;
  updatedAt: number;
  hasApiKey: boolean;
};

export function toProviderPanelRow(provider: AIProviderRow): ProviderPanelRow {
  return {
    id: provider.id,
    updatedAt: provider.updatedAt,
    hasApiKey: provider.hasApiKey,
    preset: "",
    name: provider.name,
    description: provider.description,
    kind: provider.kind,
    baseUrl: provider.baseUrl,
    model: provider.model,
    // Left blank on edit: the secret is never sent to the client, and an
    // omitted value tells the server to keep what it already has.
    apiKey: "",
    allowPrivateNetwork: provider.allowPrivateNetwork,
    structuredOutputMode: provider.structuredOutputMode,
    reasoningEffort: provider.reasoningEffort,
    extraBodyText: extraBodyToText(provider.extraBody),
    inputCostPerMillion: decimalToNumber(provider.inputCostPerMillion),
    outputCostPerMillion: decimalToNumber(provider.outputCostPerMillion),
    maxTokens: provider.maxTokens,
    tasks: provider.tasks.length > 0 ? [...provider.tasks] : null,
    priority: provider.priority,
    embeddingDimensionsChoice:
      provider.embeddingDimensions === null || provider.embeddingDimensions === undefined
        ? ""
        : String(provider.embeddingDimensions),
    embeddingInputStyle: provider.embeddingInputStyle,
    trusted: provider.trusted,
    enabled: provider.enabled,
    version: provider.version,
  };
}

/**
 * The stored object is shown as the pretty JSON a person would paste back.
 * The GraphQL JSON scalar arrives as unknown, so the shape is checked here
 * rather than asserted: anything that is not an object with keys is no
 * vendor fields at all, and an empty box says that plainly.
 */
function extraBodyToText(value: unknown): string {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    return "";
  }
  if (Object.keys(value).length === 0) {
    return "";
  }

  return JSON.stringify(value, null, 2);
}

/** The GraphQL Decimal scalar is a string; the form edits a number. */
function decimalToNumber(value: string | null | undefined): number | null {
  if (value === null || value === undefined || value === "") {
    return null;
  }
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
}
