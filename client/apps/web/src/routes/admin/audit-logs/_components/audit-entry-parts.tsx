import { audit } from "@/lib/queries/audit";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { useState } from "react";
import { entrySentence } from "./audit-entry-sentence";
import { AuditJsonBlock } from "./audit-json-block";
import {
  channelLabel,
  formatAuditValue,
  formatAuditFieldValue,
  formatFieldLabel,
  isRecordValue,
  isSensitiveOmittedValue,
  type NormalizedAuditChange,
} from "./audit-log-formatters";

/** One side of a changed field: a value, or a structure that opens as JSON on request. */
export function AuditValue({
  value,
  path,
  currency,
}: {
  value: unknown;
  path?: string;
  /** The record's currency, for a field that holds money in minor units. */
  currency?: string | null;
}) {
  const t = useT();
  const [expanded, setExpanded] = useState(false);

  if (!Array.isArray(value) && !isRecordValue(value)) {
    const formatted = formatAuditFieldValue(value, path, currency);
    return (
      <span className="flex min-w-0 flex-col gap-0.5">
        {isSensitiveOmittedValue(value) ? (
          <Badge variant="warning" className="h-4.5 self-start px-1.5 text-xs">
            {t("Sensitive")}
          </Badge>
        ) : null}
        <span className="wrap-anywhere">{formatted.value}</span>
        {formatted.transformed ? (
          <span className="text-muted-foreground font-mono text-xs">
            {t("Raw: {0}", formatAuditValue(value))}
          </span>
        ) : null}
      </span>
    );
  }

  const count = Array.isArray(value) ? value.length : Object.keys(value).length;
  return (
    <span className="flex min-w-0 flex-col gap-1.5">
      <span className="flex items-center gap-2">
        <span className="text-muted-foreground text-xs">
          {Array.isArray(value)
            ? t("{0, plural, one {List of # item} other {List of # items}}", count)
            : t("{0, plural, one {Object with # field} other {Object with # fields}}", count)}
        </span>
        <Button
          type="button"
          variant="ghost"
          size="xxs"
          onClick={() => setExpanded((current) => !current)}
        >
          {expanded ? t("Hide JSON") : t("View JSON")}
        </Button>
      </span>
      {expanded ? <AuditJsonBlock value={value} /> : null}
    </span>
  );
}

function FieldHistory({
  resourceId,
  field,
  currency,
}: {
  resourceId: string;
  field: string;
  currency: string | null;
}) {
  const t = useT();
  const { data, isPending, isError } = useQuery(audit.fieldHistory(resourceId, field));

  if (isPending) {
    return (
      <div aria-busy aria-label={t("Loading this field's history")} className="flex flex-col gap-1">
        <span aria-hidden className="ui-shimmer block h-5 rounded-sm" />
        <span aria-hidden className="ui-shimmer block h-5 rounded-sm" />
      </div>
    );
  }
  if (isError || !data || data.length === 0) {
    return (
      <p className="text-muted-foreground text-xs">{t("No earlier changes to this field.")}</p>
    );
  }

  return (
    <ol className="flex flex-col gap-1">
      {data.map((change) => {
        const value = isRecordValue(change.changes) ? change.changes[field] : undefined;
        const to = isRecordValue(value) ? value.to : undefined;
        return (
          <li key={change.id} className="flex items-baseline justify-between gap-3 text-xs">
            <span className="wrap-anywhere">
              {formatAuditFieldValue(to, field, currency).value}
            </span>
            <span className="text-muted-foreground shrink-0">
              {t(
                "{0} by {1}",
                formatToUserTimezone(change.timestamp),
                change.actorName || t("unknown"),
              )}
            </span>
          </li>
        );
      })}
    </ol>
  );
}

/** A changed field: what it was, what it became, and on request every earlier value it held. */
export function ChangeRow({
  change,
  resourceId,
  currency,
}: {
  change: NormalizedAuditChange;
  resourceId: string;
  currency: string | null;
}) {
  const t = useT();
  const [showHistory, setShowHistory] = useState(false);

  return (
    <div className="grid grid-cols-[6rem_minmax(0,1fr)] gap-x-3 border-b px-3 py-2 last:border-b-0">
      <span className="text-foreground-subtle pt-px text-xs font-medium">
        {formatFieldLabel(change.path)}
      </span>
      <div className="flex min-w-0 flex-col gap-1 text-sm">
        {change.type === "added" ? (
          <span className="text-foreground-subtle text-xs">{t("Was empty")}</span>
        ) : (
          <span className="text-muted-foreground decoration-muted-foreground/45 line-through">
            <AuditValue value={change.from} path={change.path} currency={currency} />
          </span>
        )}
        {change.type === "removed" ? (
          <span className="text-foreground-subtle text-xs">{t("Cleared")}</span>
        ) : (
          <span className="flex items-start gap-1.5">
            <span aria-hidden className="bg-success mt-1.5 size-1.5 shrink-0 rounded-full" />
            <AuditValue value={change.to} path={change.path} currency={currency} />
          </span>
        )}
        <button
          type="button"
          aria-expanded={showHistory}
          onClick={() => setShowHistory((current) => !current)}
          className="ui-focus-ring text-foreground-subtle hover:text-foreground self-start rounded-sm text-xs underline underline-offset-3"
        >
          {showHistory ? t("Hide history") : t("History")}
        </button>
        {showHistory ? (
          <div className="rounded-md border p-2">
            <FieldHistory resourceId={resourceId} field={change.path} currency={currency} />
          </div>
        ) : null}
      </div>
    </div>
  );
}

/**
 * Everything else written while serving one request, oldest first: the rows a "+N more in this
 * request" opens in the trail, and the request's panel in the inspector.
 */
export function RequestEntries({
  correlationId,
  entryId,
  onOpen,
}: {
  correlationId: string;
  entryId: string;
  onOpen?: (id: string) => void;
}) {
  const t = useT();
  const { data, isPending, isError } = useQuery(audit.requestEntries(correlationId));

  if (isPending) {
    return (
      <div aria-busy aria-label={t("Loading the other entries from this request")} className="p-2">
        <span aria-hidden className="ui-shimmer block h-10 rounded-sm" />
      </div>
    );
  }
  if (isError) {
    return (
      <p className="text-muted-foreground px-3 py-2 text-xs">
        {t("The request's other entries could not be loaded.")}
      </p>
    );
  }
  const others = (data ?? []).filter((entry) => entry.id !== entryId);
  if (others.length === 0) {
    return (
      <p className="text-muted-foreground px-3 py-2 text-xs">
        {t("This request wrote no other entry you can read.")}
      </p>
    );
  }

  return (
    <ol className="divide-y">
      {others.map((entry) => (
        <li key={entry.id}>
          <button
            type="button"
            disabled={!onOpen}
            onClick={() => onOpen?.(entry.id)}
            className="ui-focus-ring hover:bg-muted/60 grid w-full grid-cols-[minmax(0,1fr)_auto] gap-3 px-3 py-1.5 text-left text-xs disabled:cursor-default disabled:hover:bg-transparent"
          >
            <span className="truncate">
              {entrySentence(entry)}
              {entry.actorName ? (
                <span className="text-muted-foreground"> · {entry.actorName}</span>
              ) : null}
            </span>
            <span className="text-foreground-subtle shrink-0 font-mono">
              {entry.action || channelLabel(entry.channel)}
            </span>
          </button>
        </li>
      ))}
    </ol>
  );
}
