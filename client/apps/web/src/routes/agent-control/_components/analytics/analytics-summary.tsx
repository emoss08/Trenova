import type { AgentAnalytics, AIAnalyticsSegment } from "@/lib/graphql/ai-analytics";
import { Button } from "@trenova/shared/components/ui/button";
import { useNow } from "@trenova/shared/hooks/use-now";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { Hero, Ref } from "../kit/hero";
import { Ic } from "../kit/ic";

type AnalyticsSummaryProps = {
  analytics: AgentAnalytics | undefined;
  loading: boolean;
  onExport?: () => void;
  onOpenSeries: (key: string) => void;
  onOpenProvider: (providerId: string) => void;
};

/** Nova's read of the comparison: the best completer, a connection gap, a model misusing tools. */
export function AnalyticsSummary({
  analytics,
  loading,
  onExport,
  onOpenSeries,
  onOpenProvider,
}: AnalyticsSummaryProps) {
  const t = useT();
  const now = useNow("minute");
  const refreshedAt = analytics?.refreshedAt ?? null;
  const sentenceKey = (analytics?.summary ?? []).map((segment) => segment.text).join("");

  return (
    <Hero
      context={t("Analytics")}
      working={loading}
      sentenceKey={sentenceKey}
      control={
        <>
          <Button type="button" variant="outline" size="sm" disabled={!onExport} onClick={onExport}>
            <Ic n="download" s={13} />
            {t("Export")}
          </Button>
          {analytics ? (
            <span className="an-upd">
              {refreshedAt === null
                ? t("Working out the first analytics")
                : t("Updated {0}", formatRelativeTime(refreshedAt - now))}
            </span>
          ) : (
            <span className="an-upd ui-shimmer block h-3 w-28 rounded" aria-hidden />
          )}
        </>
      }
    >
      {loading ? (
        <span className="flex flex-col gap-2" aria-busy>
          <span className="block h-6 w-11/12 rounded ui-shimmer" />
          <span className="block h-6 w-2/3 rounded ui-shimmer" />
        </span>
      ) : (
        (analytics?.summary ?? []).map((segment, index) => (
          <Sentence
            key={index}
            segment={segment}
            onOpenSeries={onOpenSeries}
            onOpenProvider={onOpenProvider}
          />
        ))
      )}
    </Hero>
  );
}

type SentenceProps = {
  segment: AIAnalyticsSegment;
  onOpenSeries: (key: string) => void;
  onOpenProvider: (providerId: string) => void;
};

function Sentence({ segment, onOpenSeries, onOpenProvider }: SentenceProps) {
  const tone = segment.tone === "danger" ? "d" : segment.tone === "warn" ? "w" : undefined;
  if (segment.target === "Series" && segment.seriesKey) {
    const key = segment.seriesKey;
    return (
      <Ref tone={tone} onOpen={() => onOpenSeries(key)}>
        {segment.text}
      </Ref>
    );
  }
  if (segment.target === "Provider" && segment.providerId) {
    const id = segment.providerId;
    return (
      <Ref tone={tone} onOpen={() => onOpenProvider(id)}>
        {segment.text}
      </Ref>
    );
  }
  if (segment.strong) {
    return <b>{segment.text}</b>;
  }

  return <>{segment.text}</>;
}
