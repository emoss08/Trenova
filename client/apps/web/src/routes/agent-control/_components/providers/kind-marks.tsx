import { brandMarkFor } from "@trenova/shared/components/ui/logos/registry";
import type { ReactNode } from "react";

/**
 * The vendor whose wire protocol a kind speaks. A protocol is easier to
 * recognise by the company that published it than by its name, and every one of
 * these is somebody's published API rather than a neutral standard.
 */
const KIND_PRESET: Record<string, string> = {
  OpenAIResponses: "openai",
  OpenAIChat: "openai",
  AnthropicMessages: "anthropic",
  Ollama: "ollama",
};

export function kindMark(kind: string): ReactNode {
  const preset = KIND_PRESET[kind];
  if (!preset) {
    return null;
  }

  const Mark = brandMarkFor({ presetKey: preset });

  // No sizing class: whatever slot renders this decides the box.
  return Mark ? <Mark /> : null;
}
