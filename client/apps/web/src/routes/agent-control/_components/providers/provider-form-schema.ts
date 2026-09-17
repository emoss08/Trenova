import { aiTaskSchema, saveAIProviderRequestSchema } from "@/types/ai-provider";
import type { AIProviderRow } from "@/lib/graphql/ai-provider";
import { z } from "zod";
import type { ProviderFormValues } from "./build-save-payload";

/**
 * The save contract plus the two things only the form knows: which preset
 * was picked (never sent) and a nullable task list, which is how the checkbox
 * group represents "none selected".
 */
export const providerFormSchema = saveAIProviderRequestSchema.omit({ tasks: true }).extend({
  preset: z.string().default(""),
  tasks: z.array(aiTaskSchema).nullable().default(null),
});

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
  maxTokens: 8192,
  tasks: null,
  priority: 100,
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
    maxTokens: provider.maxTokens,
    tasks: provider.tasks.length > 0 ? [...provider.tasks] : null,
    priority: provider.priority,
    trusted: provider.trusted,
    enabled: provider.enabled,
    version: provider.version,
  };
}
