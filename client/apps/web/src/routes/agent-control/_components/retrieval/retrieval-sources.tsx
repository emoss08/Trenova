import { SectionPanel } from "@/components/section-panel";
import type {
  AIRetrievalModelChange,
  AIRetrievalSource,
  AIRetrievalSourceType,
  AIRetrievalStatus,
} from "@/lib/graphql/ai-retrieval";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Progress } from "@trenova/shared/components/ui/progress";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { RefreshCwIcon } from "lucide-react";
import {
  SOURCE_LABEL,
  SOURCE_STATE,
  modelChangeShare,
  sourceState,
  sourceWaiting,
} from "./retrieval-model";

const INDEXED_AT_FORMAT = {
  month: "short",
  day: "numeric",
  hour: "numeric",
  minute: "2-digit",
} as const;

type RetrievalSourcesProps = {
  status: AIRetrievalStatus;
  canUpdate: boolean;
  onReindex: (sourceType: AIRetrievalSourceType) => void;
  onShowFailures: (sourceType: AIRetrievalSourceType) => void;
};

/**
 * Each source's place in the index under the model searches use: how many
 * items are indexed, waiting or failed, when the last one was indexed, and a
 * re-index for when the text or the chunking changed. A model change in
 * progress is shown above them, because searches keep the old model until
 * every source is indexed under the new one.
 */
export function RetrievalSources({
  status,
  canUpdate,
  onReindex,
  onShowFailures,
}: RetrievalSourcesProps) {
  const t = useT();
  const activeModel = status.settings.activeModelKey;

  return (
    <SectionPanel
      title={t("Sources")}
      hint={activeModel ? <span className="font-mono">{activeModel}</span> : undefined}
      help={t(
        "Counts are for the model searches use now. An item is skipped on purpose when it is retired, superseded, has no text yet, or belongs to a record too sensitive to send to a provider; skipped items are still found by their words.",
      )}
    >
      {status.modelChange ? <ModelChangeProgress change={status.modelChange} /> : null}
      {!status.modelChange && status.configuredModelDiffers && status.configuredModelKey ? (
        <p className="border-border text-muted-foreground border-b px-3 py-2 text-xs">
          {t(
            "The Embedding task now routes to {0}. Every source is indexed under it before searches move to it.",
            status.configuredModelKey,
          )}
        </p>
      ) : null}
      <ul className="divide-border flex flex-col divide-y">
        {status.sources.map((source) => (
          <SourceRow
            key={source.sourceType}
            source={source}
            status={status}
            canUpdate={canUpdate}
            onReindex={onReindex}
            onShowFailures={onShowFailures}
          />
        ))}
      </ul>
    </SectionPanel>
  );
}

function ModelChangeProgress({ change }: { change: AIRetrievalModelChange }) {
  const t = useT();
  const share = modelChangeShare(change);

  return (
    <div className="border-border flex flex-col gap-1.5 border-b px-3 py-2.5">
      <div className="flex items-baseline justify-between gap-2 text-xs">
        <span className="font-medium">{t("Changing the embedding model")}</span>
        <span className="text-muted-foreground tabular-nums">{Math.round(share * 100)}%</span>
      </div>
      <Progress
        value={change.indexed}
        max={Math.max(change.total, 1)}
        size="sm"
        aria-label={t("Items indexed under the new model")}
      />
      <p className="text-muted-foreground text-xs">
        {t(
          "{0} of {1} indexed under {2}. Searches use {3} until every source is done.",
          formatNumber(change.indexed),
          formatNumber(change.total),
          change.toModelKey,
          change.fromModelKey || t("no model"),
        )}
        {change.failed > 0
          ? ` ${t("{0, plural, one {# item failed.} other {# items failed.}}", change.failed)}`
          : ""}
      </p>
    </div>
  );
}

function SourceRow({
  source,
  status,
  canUpdate,
  onReindex,
  onShowFailures,
}: {
  source: AIRetrievalSource;
  status: AIRetrievalStatus;
  canUpdate: boolean;
  onReindex: (sourceType: AIRetrievalSourceType) => void;
  onShowFailures: (sourceType: AIRetrievalSourceType) => void;
}) {
  const t = useT();
  const state = SOURCE_STATE[sourceState(source, status)];
  const waiting = sourceWaiting(source);

  return (
    <li className="flex flex-col gap-2 px-3 py-2.5 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex min-w-0 flex-col gap-0.5">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">{t(SOURCE_LABEL[source.sourceType].label)}</span>
          <Badge variant={phaseTone(state.phase)} title={state.description && t(state.description)}>
            {t(state.text)}
          </Badge>
        </div>
        <p className="text-muted-foreground text-xs tabular-nums">
          {t(
            "{0} indexed · {1} waiting · {2} failed · {3} skipped, of {4}",
            formatNumber(source.indexed),
            formatNumber(waiting),
            formatNumber(source.failed),
            formatNumber(source.skipped),
            formatNumber(source.total),
          )}
        </p>
        <p className="text-muted-foreground text-xs">
          {source.lastIndexedAt
            ? t(
                "Last indexed {0}",
                formatUnixInUserTimezone(source.lastIndexedAt, INDEXED_AT_FORMAT),
              )
            : t("Never indexed")}
        </p>
      </div>
      <div className="flex shrink-0 items-center gap-1.5">
        {source.failed > 0 ? (
          <Button variant="ghost" size="xs" onClick={() => onShowFailures(source.sourceType)}>
            {t("Show failures")}
          </Button>
        ) : null}
        {canUpdate ? (
          <Button
            variant="outline"
            size="xs"
            onClick={() => onReindex(source.sourceType)}
            disabled={!source.enabled || !status.settings.activeModelKey}
            title={
              !source.enabled
                ? t("Turn the source on to index it")
                : !status.settings.activeModelKey
                  ? t("Nothing is indexed yet")
                  : undefined
            }
          >
            <RefreshCwIcon className="size-3.5" />
            {t("Re-index")}
          </Button>
        ) : null}
      </div>
    </li>
  );
}
