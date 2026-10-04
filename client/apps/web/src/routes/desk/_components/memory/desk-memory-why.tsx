import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { memoryWhy } from "./memory-format";

/**
 * Why a memory is kept and what it replaced, under its words. Nothing is
 * drawn for a memory a person wrote down themselves.
 */
export function DeskMemoryWhy({
  memory,
  className,
}: {
  memory: Parameters<typeof memoryWhy>[0];
  className?: string;
}) {
  const t = useT();
  const lines = memoryWhy(memory, t);
  if (lines.length === 0) {
    return null;
  }

  return (
    <dl className={cn("dk-mm-why", className)}>
      {lines.map((line, index) => (
        <div key={`${line.key}-${index}`}>
          <dt>{line.label}</dt>
          <dd>{line.text}</dd>
        </div>
      ))}
    </dl>
  );
}
