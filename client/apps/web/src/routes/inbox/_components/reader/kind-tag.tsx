import type { InboundClassification } from "@/lib/graphql/inbox";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { CLASSIFICATION_DOT, classificationLabel } from "../classification";

/** A kind as a chip reads it: its categorical dot and its name, or a grey dot when not read. */
export function KindTag({
  kind,
  className,
}: {
  kind: InboundClassification | null;
  className?: string;
}) {
  const t = useT();

  return (
    <span className={cn("inline-flex items-center gap-1.5 whitespace-nowrap", className)}>
      <span
        aria-hidden
        className={cn(
          "size-2 shrink-0 rounded-sm",
          kind === null ? "bg-foreground-subtle" : CLASSIFICATION_DOT[kind],
        )}
      />
      {kind === null ? t("Not read") : classificationLabel(t, kind)}
    </span>
  );
}
