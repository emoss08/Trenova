import { StreamedText, type StreamedSegment } from "@/components/streamed-text";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

type NovaSummaryProps = {
  /** The tab the sentence heads, as it is named beside Nova. */
  context: string;
  segments: StreamedSegment[];
  /** Restarts the sentence's arrival when it changes; the same sentence never arrives twice. */
  streamKey: string;
  /** Something is working right now, so the ring turns. */
  working?: boolean;
  loading?: boolean;
  /** The one control the tab offers beside its sentence. */
  control?: ReactNode;
};

/**
 * The sentence that heads every tab of AI control: what is true now and what needs a
 * person, with links to where to act on it, and one control beside it.
 */
export function NovaSummary({
  context,
  segments,
  streamKey,
  working = false,
  loading = false,
  control,
}: NovaSummaryProps) {
  const t = useT();

  return (
    <section className="flex flex-col gap-4 border-b border-border pb-5 md:flex-row md:items-end">
      <div className="flex min-w-0 flex-1 flex-col gap-2">
        <span className="flex items-center gap-2 text-xs text-muted-foreground">
          <span
            aria-hidden
            className={cn(
              "size-4 rounded-full bg-[conic-gradient(from_0deg,var(--brand),transparent_70%)]",
              working && "motion-safe:animate-spin",
            )}
          />
          <b className="font-medium text-foreground">{t("Nova")}</b>
          <span>{context}</span>
        </span>
        {loading ? (
          <div className="flex flex-col gap-2" aria-busy>
            <Skeleton className="h-5 w-11/12" />
            <Skeleton className="h-5 w-2/3" />
          </div>
        ) : (
          <StreamedText
            segments={segments}
            streamKey={streamKey}
            className="max-w-4xl text-lg/relaxed text-muted-foreground"
          />
        )}
      </div>
      {control && <div className="flex shrink-0 flex-col items-start gap-1.5 md:items-end">{control}</div>}
    </section>
  );
}
