import type { AIProviderKind } from "@/types/ai-provider";

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
