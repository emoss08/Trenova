import type { AIProviderRow } from "@/lib/graphql/ai-provider";
import type { AIProviderKind, AITask, AITaskDescriptor } from "@/types/ai-provider";
import { EMBEDDING_DIMENSIONS } from "@/types/ai-provider";
import { kindSupportsEmbedding } from "./provider-form-schema";

/** What decides where a task goes: the fields routing reads, of a saved provider or a draft. */
export type RoutingProvider = Pick<
  AIProviderRow,
  "id" | "name" | "kind" | "tasks" | "trusted" | "enabled" | "hasApiKey" | "embeddingDimensions"
>;

/** A task as the routing grid shows it: its name, whether it needs trust, and what happens without a provider. */
export type TaskMeta = {
  task: AITask;
  label: string;
  trust: boolean;
};

/** Why an assigned provider is passed over for a task. */
export type SkipReason = "off" | "needsKey" | "untrusted" | "noEmbedding";

export type Route<P extends RoutingProvider = RoutingProvider> = {
  first: P | null;
  next: P | null;
  assigned: P[];
};

/**
 * The tasks in the order the routing grid lists them, from the catalog: what a person
 * reads in Trenova first, the background work after, and General last because it takes
 * whatever no other task names.
 */
const TASK_ORDER: readonly AITask[] = [
  "AssistantChat",
  "BillingDiagnosis",
  "DocumentExtraction",
  "DocumentClassification",
  "ScopeClassification",
  "OperationalInsights",
  "DailyBriefing",
  "FormulaAssistant",
  "QueryCompose",
  "InboundClassification",
  "AccountingMapping",
  "EvaluationJudge",
  "Embedding",
  "General",
];

export function taskMetas(descriptors: readonly AITaskDescriptor[]): TaskMeta[] {
  const byTask = new Map(descriptors.map((descriptor) => [descriptor.task, descriptor]));
  return TASK_ORDER.flatMap((task) => {
    const descriptor = byTask.get(task);
    return descriptor
      ? [{ task, label: descriptor.label, trust: descriptor.requiresTrust }]
      : [];
  });
}

/** Whether a provider's protocol can only be reached with a key, and it has none. */
export function needsKey(
  provider: Pick<RoutingProvider, "kind" | "hasApiKey">,
  keyRequired: (kind: AIProviderKind) => boolean,
): boolean {
  return keyRequired(provider.kind) && !provider.hasApiKey;
}

/**
 * Why a provider assigned to a task does not take it, or null when it does. This is the
 * server's rule for serving a task (`Provider.CanServeTask`), in the same order, so the
 * grid never claims a route the router would refuse.
 */
export function skipReason(
  provider: RoutingProvider,
  meta: TaskMeta,
  keyRequired: (kind: AIProviderKind) => boolean,
): SkipReason | null {
  if (!provider.enabled) {
    return needsKey(provider, keyRequired) ? "needsKey" : "off";
  }
  if (meta.trust && !provider.trusted) {
    return "untrusted";
  }
  if (meta.task === "Embedding") {
    const dimensions = provider.embeddingDimensions ?? 0;
    if (
      !kindSupportsEmbedding(provider.kind) ||
      !(EMBEDDING_DIMENSIONS as readonly number[]).includes(dimensions)
    ) {
      return "noEmbedding";
    }
  }
  return null;
}

/** Where one task goes: the first provider in order that takes it, and the one after. */
export function routeOf<P extends RoutingProvider>(
  meta: TaskMeta,
  providers: readonly P[],
  keyRequired: (kind: AIProviderKind) => boolean,
): Route<P> {
  const assigned = providers.filter((provider) => provider.tasks.includes(meta.task));
  const taking = assigned.filter((provider) => skipReason(provider, meta, keyRequired) === null);
  return { first: taking[0] ?? null, next: taking[1] ?? null, assigned };
}

/** The tasks a provider takes first, in grid order. */
export function firstsOf<P extends RoutingProvider>(
  provider: P,
  metas: readonly TaskMeta[],
  providers: readonly P[],
  keyRequired: (kind: AIProviderKind) => boolean,
): TaskMeta[] {
  return metas.filter((meta) => routeOf(meta, providers, keyRequired).first?.id === provider.id);
}

/** The tasks no provider takes. */
export function uncovered(
  metas: readonly TaskMeta[],
  providers: readonly RoutingProvider[],
  keyRequired: (kind: AIProviderKind) => boolean,
): TaskMeta[] {
  return metas.filter((meta) => routeOf(meta, providers, keyRequired).first === null);
}

/** A task assigned or unassigned, keeping the order the provider already lists its tasks in. */
export function toggleTask(tasks: readonly AITask[], task: AITask): AITask[] {
  return tasks.includes(task) ? tasks.filter((entry) => entry !== task) : [...tasks, task];
}

/** The order with one provider moved by a step, or the same order at either end. */
export function moveBy(ids: readonly string[], id: string, delta: -1 | 1): string[] {
  const from = ids.indexOf(id);
  const to = from + delta;
  if (from < 0 || to < 0 || to >= ids.length) {
    return [...ids];
  }
  const next = [...ids];
  next.splice(from, 1);
  next.splice(to, 0, id);
  return next;
}

/** The order with one provider dropped where another stands. */
export function moveTo(ids: readonly string[], id: string, over: string): string[] {
  const from = ids.indexOf(id);
  const to = ids.indexOf(over);
  if (from < 0 || to < 0 || from === to) {
    return [...ids];
  }
  const next = [...ids];
  next.splice(from, 1);
  next.splice(to, 0, id);
  return next;
}

export function sameOrder(a: readonly string[], b: readonly string[]): boolean {
  return a.length === b.length && a.every((id, index) => id === b[index]);
}

/** A provider's connection, as its dot shows it: testing now, last answered, last failed, never tried. */
export type LiveState = "run" | "ok" | "fail" | "none";

export function liveState(
  lastTest: AIProviderRow["lastTest"],
  testing: boolean,
  recentFailure: boolean,
): LiveState {
  if (testing) {
    return "run";
  }
  if (recentFailure) {
    return "fail";
  }
  if (!lastTest) {
    return "none";
  }
  return lastTest.success ? "ok" : "fail";
}

/** Milliseconds as the chain reads them: 410ms, 1.4s. */
export function formatLatency(ms: number): string {
  return ms >= 1000 ? `${(ms / 1000).toFixed(1)}s` : `${Math.round(ms)}ms`;
}

/** Tokens as the read sheet reads them: 820, 14.2k, 1.4M. */
export function formatTokens(tokens: number): string {
  if (tokens >= 1_000_000) {
    return `${(tokens / 1_000_000).toFixed(1)}M`;
  }
  if (tokens >= 10_000) {
    return `${Math.round(tokens / 1000)}k`;
  }
  if (tokens >= 1000) {
    return `${(tokens / 1000).toFixed(1)}k`;
  }
  return String(tokens);
}

/** A loopback, link-local, RFC 1918 or single-label host: one only Private network reaches. */
const PRIVATE_HOST =
  /^(localhost|127\.|10\.|192\.168\.|169\.254\.|172\.(1[6-9]|2\d|3[01])\.|0\.0\.0\.0|\[?::1\]?$|\[?f[cd][0-9a-f]{2}:)/i;

export function hostOf(url: string): string | null {
  try {
    return new URL(url.trim()).hostname.toLowerCase();
  } catch {
    return null;
  }
}

export function isPrivateAddress(url: string): boolean {
  const host = hostOf(url);
  if (!host) {
    return false;
  }
  return (
    PRIVATE_HOST.test(host) ||
    !host.includes(".") ||
    host.endsWith(".local") ||
    host.endsWith(".internal") ||
    host.endsWith(".lan")
  );
}

/** What is wrong with a base URL as typed, before anything is asked of it. */
export type BaseUrlProblem = "scheme" | "private" | null;

export function baseUrlProblem(baseUrl: string, allowPrivateNetwork: boolean): BaseUrlProblem {
  const trimmed = baseUrl.trim();
  if (trimmed === "") {
    return null;
  }
  if (!/^https?:\/\//i.test(trimmed) || hostOf(trimmed) === null) {
    return "scheme";
  }
  if (isPrivateAddress(trimmed) && !allowPrivateNetwork) {
    return "private";
  }
  return null;
}

/** The key's start a vendor documents, used as the key field's placeholder. */
export function keyPlaceholder(kind: AIProviderKind, baseUrl: string): string | null {
  const host = hostOf(baseUrl) ?? "";
  if (host.includes("groq")) return "gsk_…";
  if (host.includes("openrouter")) return "sk-or-…";
  if (kind === "AnthropicMessages" && (host === "" || host.endsWith("anthropic.com"))) {
    return "sk-ant-…";
  }
  if (kind === "OpenAIResponses" && (host === "" || host.endsWith("openai.com"))) return "sk-…";
  return null;
}
