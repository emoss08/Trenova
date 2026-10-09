import type {
  AIRetrievalModelChange,
  AIRetrievalSource,
  AIRetrievalSourceType,
  AIRetrievalStatus,
} from "@/lib/graphql/ai-retrieval";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Ic, type IcName } from "../kit/ic";
import { SecH, Switch } from "../kit/layout";
import {
  SOURCE_LABEL,
  SOURCE_STATE,
  SOURCE_STATE_TONE,
  modelChangeShare,
  sourceShares,
  sourceState,
  sourceWaiting,
} from "./retrieval-model";
import { Button } from "@trenova/shared/components/ui/button";

const SOURCE_ICON: Record<AIRetrievalSourceType, IcName> = {
  Memory: "brain",
  Document: "receipt",
  InboundMessage: "inbox",
};

type RetrievalSourcesProps = {
  status: AIRetrievalStatus;
  canUpdate: boolean;
  busy: boolean;
  onToggle: (source: AIRetrievalSourceType, enabled: boolean) => void;
  onReindex: (source: AIRetrievalSourceType) => void;
  onShowFailures: (source: AIRetrievalSourceType) => void;
};

/**
 * Each source's place in the index under the model searches use: how much is indexed,
 * waiting, failed or skipped, a switch to index it at all, and a re-index for when its
 * text or the chunking changed. A model change in progress sits above them, because
 * searches keep the old model until every source is indexed under the new one.
 */
export function RetrievalSources({
  status,
  canUpdate,
  busy,
  onToggle,
  onReindex,
  onShowFailures,
}: RetrievalSourcesProps) {
  const t = useT();
  const model = status.settings.activeModelKey;

  return (
    <section className="sec">
      <SecH t={t("Sources")} r={model ? <span className="sh2-n mono">{model}</span> : null} />
      {status.modelChange && <ModelChange change={status.modelChange} />}
      {!status.modelChange && status.configuredModelDiffers && status.configuredModelKey && (
        <p className="lead">
          {t(
            "The Embedding task now routes to {0}. Every source is indexed under it before searches move to it.",
            status.configuredModelKey,
          )}
        </p>
      )}
      <div className="srcs">
        {status.sources.map((source) => (
          <SourceRow
            key={source.sourceType}
            source={source}
            status={status}
            canUpdate={canUpdate}
            busy={busy}
            onToggle={onToggle}
            onReindex={onReindex}
            onShowFailures={onShowFailures}
          />
        ))}
      </div>
    </section>
  );
}

function ModelChange({ change }: { change: AIRetrievalModelChange }) {
  const t = useT();
  const share = modelChangeShare(change);

  return (
    <div className="src">
      <span className="src-i">
        <Ic n="refresh" s={15} />
      </span>
      <div className="src-m">
        <div className="src-h">
          <b>{t("Changing the embedding model")}</b>
          <span className="tg b">
            <i className="spn" />
            {Math.round(share * 100)}%
          </span>
        </div>
        <p>
          {t(
            "{0} of {1} indexed under {2}. Searches use {3} until every source is done.",
            formatNumber(change.indexed),
            formatNumber(change.total),
            change.toModelKey,
            change.fromModelKey || t("no model"),
          )}
          {change.failed > 0 &&
            ` ${t("{0, plural, one {# item failed.} other {# items failed.}}", change.failed)}`}
        </p>
        <div
          className="sbar"
          role="progressbar"
          aria-label={t("Items indexed under the new model")}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={Math.round(share * 100)}
        >
          <i className="ix" style={{ width: `${share * 100}%` }} />
        </div>
      </div>
    </div>
  );
}

type SourceRowProps = Omit<RetrievalSourcesProps, "status"> & {
  source: AIRetrievalSource;
  status: AIRetrievalStatus;
};

function SourceRow({
  source,
  status,
  canUpdate,
  busy,
  onToggle,
  onReindex,
  onShowFailures,
}: SourceRowProps) {
  const t = useT();
  const stateKey = sourceState(source, status);
  const state = SOURCE_STATE[stateKey];
  const waiting = sourceWaiting(source);
  const shares = sourceShares(source);
  const label = t(SOURCE_LABEL[source.sourceType].label);
  const indexed = Boolean(status.settings.activeModelKey);

  return (
    <div className={cn("src", !source.enabled && "dim")}>
      <span className="src-i">
        <Ic n={SOURCE_ICON[source.sourceType]} s={15} />
      </span>
      <div className="src-m">
        <div className="src-h">
          <b>{label}</b>
          <span
            className={cn("tg", SOURCE_STATE_TONE[stateKey])}
            title={state.description ? t(state.description) : undefined}
          >
            {stateKey === "indexing" && <i className="spn" />}
            {t(state.text)}
          </span>
        </div>
        <p>{t(SOURCE_LABEL[source.sourceType].description)}</p>
        <div className="sbar" aria-hidden>
          <i className="ix" style={{ width: `${shares.indexed}%` }} />
          <i className="fx" style={{ width: `${shares.failed}%` }} />
          <i className="sk" style={{ width: `${shares.skipped}%` }} />
        </div>
        <div className="src-n mono">
          <span>{t("{0} indexed", formatNumber(source.indexed))}</span>
          <span>{t("{0} waiting", formatNumber(waiting))}</span>
          {source.failed > 0 && (
            <button type="button" className="t-d" onClick={() => onShowFailures(source.sourceType)}>
              {t("{0} failed", formatNumber(source.failed))}
            </button>
          )}
          {source.skipped > 0 && (
            <span
              title={t(
                "Retired, superseded, empty, or too sensitive to send. Still found by their words.",
              )}
            >
              {t("{0} skipped", formatNumber(source.skipped))}
            </span>
          )}
          <span className="dim">{t("of {0}", formatNumber(source.total))}</span>
        </div>
      </div>
      {canUpdate && (
        <div className="src-c">
          <Button
            type="button"
            variant="outline" size="sm"
            disabled={!source.enabled || !indexed}
            title={
              !indexed
                ? t("Nothing is indexed yet")
                : !source.enabled
                  ? t("Turn the source on to index it")
                  : undefined
            }
            onClick={() => onReindex(source.sourceType)}
          >
            <Ic n="refresh" s={12} />
            {t("Re-index")}
          </Button>
          <Switch
            on={source.enabled}
            label={t("Index {0}", label)}
            disabled={busy}
            onChange={(enabled) => onToggle(source.sourceType, enabled)}
          />
        </div>
      )}
    </div>
  );
}
