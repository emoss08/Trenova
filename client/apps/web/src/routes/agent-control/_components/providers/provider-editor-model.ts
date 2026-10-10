import type {
  AIProviderDraftTest,
  AIProviderRoutingDraft,
  AIProviderRow,
  AIProviderWithLimits,
} from "@/lib/graphql/ai-provider";
import {
  aiProviderKindSchema,
  aiTaskSchema,
  capActionSchema,
  embeddingInputStyleSchema,
  EMBEDDING_DIMENSIONS,
  PROVIDER_MAX_CONCURRENT,
  PROVIDER_TIMEOUT_SECONDS,
  reasoningEffortSchema,
  structuredOutputModeSchema,
  thinkingStyleSchema,
  type AIProviderKind,
  type AIProviderPreset,
  type SaveAIProviderRequest,
} from "@/types/ai-provider";
import { translate } from "@trenova/shared/i18n/runtime";
import { z } from "zod";
import { buildSavePayload, parseEmbeddingDimensions } from "./build-save-payload";
import { embeddingIssues } from "./provider-form-schema";
import { baseUrlProblem } from "./provider-model";

/** The step the server leaves between priorities, so a new provider joins the end of the chain. */
export const PRIORITY_STEP = 10;

const DECIMAL = /^\d{0,6}(\.\d{0,6})?$/;
const WHOLE = /^\d+$/;

const decimalText = (message: () => string) =>
  z
    .string()
    .trim()
    .refine((value) => value === "" || (DECIMAL.test(value) && value !== "."), { error: message });

const wholeIn = (range: { min: number; max: number }, message: () => string) =>
  z
    .string()
    .trim()
    .refine(
      (value) => WHOLE.test(value) && Number(value) >= range.min && Number(value) <= range.max,
      { error: message },
    );

export const providerEditorSchema = z
  .object({
    /** The preset a new provider starts from; editing a saved one leaves it empty. */
    preset: z.string(),
    name: z
      .string()
      .trim()
      .min(1, { error: () => translate("Give it a name") }),
    description: z.string(),
    kind: aiProviderKindSchema,
    baseUrl: z.string(),
    model: z
      .string()
      .trim()
      .min(1, { error: () => translate("Pick a model") }),
    apiKey: z.string(),
    keepPreviousKey: z.boolean(),
    tasks: z.array(aiTaskSchema),
    trusted: z.boolean(),
    allowPrivateNetwork: z.boolean(),
    timeoutSeconds: wholeIn(PROVIDER_TIMEOUT_SECONDS, () =>
      translate(
        "Between {0} and {1} seconds",
        PROVIDER_TIMEOUT_SECONDS.min,
        PROVIDER_TIMEOUT_SECONDS.max,
      ),
    ),
    maxConcurrent: wholeIn(PROVIDER_MAX_CONCURRENT, () =>
      translate("Between {0} and {1}", PROVIDER_MAX_CONCURRENT.min, PROVIDER_MAX_CONCURRENT.max),
    ),
    monthlyCapUsd: decimalText(() => translate("A cap is an amount in dollars")).refine(
      (value) => value === "" || Number(value) > 0,
      { error: () => translate("A cap must be more than $0") },
    ),
    onCap: capActionSchema,
    inputCostPerMillion: decimalText(() => translate("A price is an amount in dollars")),
    outputCostPerMillion: decimalText(() => translate("A price is an amount in dollars")),
    cacheReadCostPerMillion: decimalText(() => translate("A price is an amount in dollars")),
    cacheWriteCostPerMillion: decimalText(() => translate("A price is an amount in dollars")),
    structuredOutputMode: structuredOutputModeSchema,
    reasoningEffort: reasoningEffortSchema,
    thinkingStyle: thinkingStyleSchema,
    maxTokens: wholeIn({ min: 256, max: 200_000 }, () =>
      translate("Between {0} and {1} tokens", 256, 200_000),
    ),
    embeddingDimensionsChoice: z
      .string()
      .refine(
        (value) =>
          value === "" || (EMBEDDING_DIMENSIONS as readonly number[]).includes(Number(value)),
        { error: () => translate("Embedding dimensions must be 768, 1024 or 1536") },
      ),
    embeddingInputStyle: embeddingInputStyleSchema,
    extraBodyText: z.string().superRefine((value, ctx) => {
      const trimmed = value.trim();
      if (trimmed === "") return;
      let parsed: unknown;
      try {
        parsed = JSON.parse(trimmed);
      } catch {
        ctx.addIssue({ code: "custom", message: translate("This is not valid JSON") });
        return;
      }
      if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
        ctx.addIssue({
          code: "custom",
          message: translate('Extra fields must be a JSON object, like {"top_k": 40}'),
        });
      }
    }),
    priority: z.number().int().min(0),
    enabled: z.boolean(),
    version: z.number(),
  })
  .superRefine((values, ctx) => {
    if (baseUrlProblem(values.baseUrl, values.allowPrivateNetwork) === "scheme") {
      ctx.addIssue({
        code: "custom",
        path: ["baseUrl"],
        message: translate("Start with http:// or https://"),
      });
    }
    for (const issue of embeddingIssues({ ...values, tasks: values.tasks })) {
      ctx.addIssue({ code: "custom", path: [issue.path], message: translate(issue.message) });
    }
  });

export type ProviderEditorValues = z.infer<typeof providerEditorSchema>;

function decimalToText(value: string | null | undefined): string {
  if (value === null || value === undefined || value === "") return "";
  const parsed = Number(value);
  return Number.isFinite(parsed) ? String(parsed) : "";
}

function extraBodyToText(value: unknown): string {
  if (value === null || typeof value !== "object" || Array.isArray(value)) return "";
  return Object.keys(value).length > 0 ? JSON.stringify(value, null, 2) : "";
}

/** A saved provider as its editor opens it. The key field starts empty: the secret never comes back. */
export function editorValuesFromProvider(provider: AIProviderWithLimits): ProviderEditorValues {
  return {
    preset: "",
    name: provider.name,
    description: provider.description,
    kind: provider.kind,
    baseUrl: provider.baseUrl,
    model: provider.model,
    apiKey: "",
    keepPreviousKey: true,
    tasks: [...provider.tasks],
    trusted: provider.trusted,
    allowPrivateNetwork: provider.allowPrivateNetwork,
    timeoutSeconds: String(provider.timeoutSeconds),
    maxConcurrent: String(provider.maxConcurrent),
    monthlyCapUsd: decimalToText(provider.monthlyCapUsd),
    onCap: provider.onCap,
    inputCostPerMillion: decimalToText(provider.inputCostPerMillion),
    outputCostPerMillion: decimalToText(provider.outputCostPerMillion),
    cacheReadCostPerMillion: decimalToText(provider.cacheReadCostPerMillion),
    cacheWriteCostPerMillion: decimalToText(provider.cacheWriteCostPerMillion),
    structuredOutputMode: provider.structuredOutputMode,
    reasoningEffort: provider.reasoningEffort,
    thinkingStyle: provider.thinkingStyle,
    maxTokens: String(provider.maxTokens),
    embeddingDimensionsChoice:
      provider.embeddingDimensions === null || provider.embeddingDimensions === undefined
        ? ""
        : String(provider.embeddingDimensions),
    embeddingInputStyle: provider.embeddingInputStyle,
    extraBodyText: extraBodyToText(provider.extraBody),
    priority: provider.priority,
    enabled: provider.enabled,
    version: provider.version,
  };
}

/** The name a new provider from a preset starts with: the vendor's, or what a local one is. */
function presetName(preset: AIProviderPreset): string {
  const vendor = preset.label.replace(/\s*\(self-hosted\)$/i, "");
  return preset.selfHosted ? translate("Local {0}", vendor) : vendor;
}

/** A new provider as a preset starts it, at the end of the chain. */
export function editorValuesFromPreset(
  preset: AIProviderPreset,
  providers: readonly Pick<AIProviderRow, "priority">[],
): ProviderEditorValues {
  const last = providers.reduce((most, provider) => Math.max(most, provider.priority), 0);
  return {
    preset: preset.key,
    name: presetName(preset),
    description: "",
    kind: preset.kind,
    baseUrl: preset.baseUrl,
    model: preset.exampleModel,
    apiKey: "",
    keepPreviousKey: false,
    tasks: [...preset.tasks],
    trusted: preset.kind === "AnthropicMessages" && !preset.selfHosted,
    allowPrivateNetwork: preset.allowPrivateNetwork,
    timeoutSeconds: String(PROVIDER_TIMEOUT_SECONDS.initial),
    maxConcurrent: String(PROVIDER_MAX_CONCURRENT.initial),
    monthlyCapUsd: "",
    onCap: "Next",
    inputCostPerMillion: "",
    outputCostPerMillion: "",
    cacheReadCostPerMillion: "",
    cacheWriteCostPerMillion: "",
    structuredOutputMode: preset.structuredOutputMode,
    reasoningEffort: "Off",
    thinkingStyle: "Auto",
    maxTokens: "8192",
    embeddingDimensionsChoice:
      preset.embeddingDimensions > 0 ? String(preset.embeddingDimensions) : "",
    embeddingInputStyle: preset.embeddingInputStyle ?? "None",
    extraBodyText: "",
    priority: last + PRIORITY_STEP,
    enabled: true,
    version: 0,
  };
}

/** The first thing stopping the editor from saving, in the order a person would fix them. */
export type EditorProblem = {
  field: "name" | "baseUrl" | "apiKey" | "model";
  message: string;
};

/** What keeps a draft from saving, named by the field that has to change. */
export function editorProblem(
  values: ProviderEditorValues,
  context: {
    create: boolean;
    keyRequired: boolean;
    hasStoredKey: boolean;
    requiresBaseUrl?: boolean;
  },
): EditorProblem | null {
  if (values.name.trim() === "") return { field: "name", message: translate("Give it a name") };
  if (context.requiresBaseUrl && values.baseUrl.trim() === "") {
    return { field: "baseUrl", message: translate("Add the base URL") };
  }
  if (baseUrlProblem(values.baseUrl, values.allowPrivateNetwork) !== null) {
    return { field: "baseUrl", message: translate("Fix the base URL") };
  }
  if (context.keyRequired && !context.hasStoredKey && values.apiKey.trim() === "") {
    return { field: "apiKey", message: translate("Paste an API key") };
  }
  if (values.model.trim() === "") return { field: "model", message: translate("Pick a model") };
  return null;
}

export function editorBlocker(
  values: ProviderEditorValues,
  context: { create: boolean; keyRequired: boolean; hasStoredKey: boolean },
): string | null {
  return editorProblem(values, context)?.message ?? null;
}

const toDecimal = (text: string): number | null => (text.trim() === "" ? null : Number(text));

/**
 * The multiples of the input price the server charges for a cached prompt
 * token when no cache price is entered (aiprovider.Kind.CacheReadMultiple and
 * CacheWriteMultiple). Only Anthropic Messages reports cache writes.
 */
const CACHE_READ_MULTIPLE = 0.1;
const ANTHROPIC_CACHE_WRITE_MULTIPLE = 1.25;

export function reportsCacheWrites(kind: ProviderEditorValues["kind"]): boolean {
  return kind === "AnthropicMessages";
}

/**
 * What a cached prompt token costs per million when its price is left empty,
 * as text for the field's placeholder; blank while the input price is.
 */
export function cachePriceDefault(
  kind: ProviderEditorValues["kind"],
  inputPrice: string,
  part: "read" | "write",
): string {
  const input = inputPrice.trim() === "" ? Number.NaN : Number(inputPrice);
  if (!Number.isFinite(input) || input < 0) {
    return "";
  }
  const multiple =
    part === "read"
      ? CACHE_READ_MULTIPLE
      : reportsCacheWrites(kind)
        ? ANTHROPIC_CACHE_WRITE_MULTIPLE
        : 1;

  return String(Number((input * multiple).toFixed(6)));
}

/**
 * The save request. A blank key on an edit means "keep the stored one"; a new provider is
 * saved off unless a test has just passed, so nothing untried starts taking work.
 */
export function toSaveRequest(
  values: ProviderEditorValues,
  context: { editing: boolean; enable: boolean },
): SaveAIProviderRequest {
  const replacing = context.editing && values.apiKey.trim() !== "";
  return {
    ...buildSavePayload(
      {
        preset: "",
        name: values.name.trim(),
        description: values.description,
        kind: values.kind,
        baseUrl: values.baseUrl.trim(),
        model: values.model.trim(),
        apiKey: values.apiKey,
        allowPrivateNetwork: values.allowPrivateNetwork,
        structuredOutputMode: values.structuredOutputMode,
        reasoningEffort: values.reasoningEffort,
        thinkingStyle: values.thinkingStyle,
        extraBodyText: values.extraBodyText,
        inputCostPerMillion: toDecimal(values.inputCostPerMillion),
        outputCostPerMillion: toDecimal(values.outputCostPerMillion),
        cacheReadCostPerMillion: toDecimal(values.cacheReadCostPerMillion),
        cacheWriteCostPerMillion: toDecimal(values.cacheWriteCostPerMillion),
        maxTokens: Number(values.maxTokens),
        tasks: values.tasks,
        priority: values.priority,
        embeddingDimensionsChoice: values.embeddingDimensionsChoice,
        embeddingInputStyle: values.embeddingInputStyle,
        trusted: values.trusted,
        enabled: context.editing ? values.enabled : context.enable,
        timeoutSeconds: Number(values.timeoutSeconds),
        maxConcurrent: Number(values.maxConcurrent),
        monthlyCapUsd: toDecimal(values.monthlyCapUsd),
        onCap: values.onCap,
        keepPreviousKey: false,
        version: values.version,
      },
      context.editing,
    ),
    keepPreviousKey: replacing && values.keepPreviousKey,
  };
}

/** The editor as the route preview reads it. */
export function routingDraft(
  values: ProviderEditorValues,
  providerId: string | null,
): AIProviderRoutingDraft {
  return {
    id: providerId,
    name: values.name.trim() || translate("New provider"),
    kind: values.kind,
    tasks: [...values.tasks],
    priority: values.priority,
    embeddingDimensions: values.tasks.includes("Embedding")
      ? parseEmbeddingDimensions(values.embeddingDimensionsChoice)
      : null,
    trusted: values.trusted,
    enabled: values.enabled,
  };
}

/** The editor as a draft test sends it. With a saved provider and no new key, the stored key is used. */
export function draftTestInput(
  values: ProviderEditorValues,
  providerId: string | null,
): AIProviderDraftTest {
  const key = values.apiKey.trim();
  const timeout = Number(values.timeoutSeconds);
  return {
    endpoint: {
      providerId,
      kind: values.kind,
      baseUrl: values.baseUrl.trim(),
      apiKey: key === "" ? null : key,
      allowPrivateNetwork: values.allowPrivateNetwork,
    },
    model: values.model.trim(),
    structuredOutputMode: values.structuredOutputMode,
    tasks: [...values.tasks],
    embeddingDimensions: values.tasks.includes("Embedding")
      ? parseEmbeddingDimensions(values.embeddingDimensionsChoice)
      : null,
    embeddingInputStyle: values.embeddingInputStyle,
    timeoutSeconds: Number.isInteger(timeout) ? timeout : null,
  };
}

/** Whether a protocol's endpoint is reached only with a key. */
export type KeyRule = (kind: AIProviderKind) => boolean;
