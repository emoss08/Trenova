import type { AgentMemoryKind } from "@trenova/graphql/generated/graphql";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

/** The swatch each kind of memory is marked with: a category, not a severity. */
export const MEMORY_KIND_TONE: Record<AgentMemoryKind, "b" | "t" | "w" | "v"> = {
  Instruction: "b",
  Fact: "t",
  Correction: "w",
  Procedure: "v",
};

export const MEMORY_KIND_LABELS: Record<AgentMemoryKind, string> = defineLabels({
  Instruction: "Instruction",
  Fact: "Fact",
  Correction: "Correction",
  Procedure: "Procedure",
});

export const MEMORY_KINDS = Object.keys(MEMORY_KIND_TONE) as AgentMemoryKind[];

/** A memory's kind as a swatch and its name. */
export function MemoryKindMark({ kind }: { kind: AgentMemoryKind }) {
  const t = useT();

  return <span className={cn("mk-k", MEMORY_KIND_TONE[kind])}>{t(MEMORY_KIND_LABELS[kind])}</span>;
}
