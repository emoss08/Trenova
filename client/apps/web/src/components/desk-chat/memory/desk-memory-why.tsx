import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { memoryWhy } from "./memory-format";

/**
 * Why a memory is kept and what it replaced, under its words. Nothing is
 * drawn for a memory a person wrote down themselves. Given `before`, the
 * memory it replaced is labelled Before and drawn by the caller, as the
 * conversation shows it: struck-through steps rather than a quote.
 */
export function DeskMemoryWhy({
  memory,
  className,
  before,
}: {
  memory: Parameters<typeof memoryWhy>[0];
  className?: string;
  before?: (content: string) => ReactNode;
}) {
  const t = useT();
  const lines = memoryWhy(memory, t);
  if (lines.length === 0) {
    return null;
  }

  return (
    <dl className={cn("dk-mm-why", className)}>
      {lines.map((line, index) => {
        const replaced = before && line.key === "replaces" ? memory.replaces : null;
        return (
          <div key={`${line.key}-${index}`}>
            <dt>{replaced ? t("Before") : line.label}</dt>
            <dd>{replaced ? before?.(replaced.content) : line.text}</dd>
          </div>
        );
      })}
    </dl>
  );
}
