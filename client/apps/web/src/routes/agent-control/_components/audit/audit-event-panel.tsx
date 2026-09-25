import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { CopyIconButton } from "@/components/copy-icon-button";
import { SectionPanel } from "@/components/section-panel";
import { panelSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { formatLatency, formatTokens, formatUsd } from "@/lib/ai-usage-format";
import type { AIAuditEventDetail, AIAuditEventRow } from "@/lib/graphql/ai-audit";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { CircleAlertIcon } from "lucide-react";
import { useQueryStates } from "nuqs";
import type { ReactNode } from "react";
import { HeldByChips } from "../safety/safety-badges";
import { TraceLink } from "../trace-link";
import { AuditOutcomeBadge } from "./audit-badges";
import {
  auditEventRecordPath,
  auditKindLabel,
  auditTierLabel,
  tierSourceLabel,
} from "./audit-model";

/**
 * One event of the trail, read in full: who it was for and who decided it,
 * what was called and why it ran at the tier it did, what it changed, what it
 * had read, the model call behind it, the arguments as the reader may see
 * them, its trace, and its place in the signed chain.
 */
export function AuditEventPanel({ open, onOpenChange, row }: DataTablePanelProps<AIAuditEventRow>) {
  const t = useT();
  const [{ panelEntityId }] = useQueryStates(panelSearchParamsParser);
  const id = row?.id ?? panelEntityId ?? "";
  const detail = useQuery({ ...queries.aiAudit.event(id), enabled: open && id !== "" });
  const event = detail.data ?? null;

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={event ? auditKindLabel(t, event.kind) : row ? auditKindLabel(t, row.kind) : t("Event")}
      description={
        event
          ? t("#{0} · {1}", event.seq.toLocaleString(), formatUnixDateTimeMedium(event.occurredAt))
          : undefined
      }
      size="lg"
    >
      {detail.isError ? (
        <Alert variant="destructive" size="sm">
          <CircleAlertIcon />
          <AlertDescription>
            {t("This event could not be loaded. Try again shortly.")}
          </AlertDescription>
        </Alert>
      ) : detail.isSuccess && event === null ? (
        <Alert size="sm">
          <CircleAlertIcon />
          <AlertDescription>
            {t("This event is not on your organization's trail, or retention has removed it.")}
          </AlertDescription>
        </Alert>
      ) : event ? (
        <AuditEventDetailSections event={event} />
      ) : (
        <div className="flex flex-col gap-3" aria-busy>
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
          <Skeleton className="h-40" />
        </div>
      )}
    </DataTablePanelContainer>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <SectionPanel title={title}>
      <div className="p-3">{children}</div>
    </SectionPanel>
  );
}

function Mono({ value }: { value: string | null | undefined }) {
  if (!value) {
    return <DescriptionEmpty />;
  }
  return <span className="font-mono text-xs break-all">{value}</span>;
}

function CopyableMono({ value, label }: { value: string | null | undefined; label: string }) {
  if (!value) {
    return <DescriptionEmpty />;
  }
  return (
    <span className="flex min-w-0 items-start gap-1">
      <span className="min-w-0 font-mono text-xs break-all">{value}</span>
      <CopyIconButton value={value} label={label} size="icon-xxs" />
    </span>
  );
}

function orEmpty(value: ReactNode | null | undefined): ReactNode {
  return value === null || value === undefined || value === "" ? <DescriptionEmpty /> : value;
}

function yesNo(t: TranslateFn, value: boolean): string {
  return value ? t("Yes") : t("No");
}

export function AuditEventDetailSections({ event }: { event: AIAuditEventDetail }) {
  const t = useT();
  const recordHref = auditEventRecordPath(event.entityType, event.entityId);

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <AuditOutcomeBadge outcome={event.outcome} />
        {event.purpose === "Evaluation" ? (
          <Badge variant="accent-violet">{t("Evaluation")}</Badge>
        ) : null}
        {event.simulated ? <Badge variant="neutral">{t("Simulated")}</Badge> : null}
        {event.reconstructed ? (
          <Badge variant="neutral" appearance="outline">
            {t("Reconstructed")}
          </Badge>
        ) : null}
      </div>

      <Section title={t("Who")}>
        <DescriptionList layout="inline">
          <DescriptionItem label={t("On behalf of")}>
            {orEmpty(event.onBehalfOf?.name ?? event.onBehalfOfUserName)}
          </DescriptionItem>
          <DescriptionItem label={t("Decided by")}>
            {orEmpty(event.decidedBy?.name ?? event.decidedByUserName)}
          </DescriptionItem>
          <DescriptionItem label={t("Acting as")}>
            {event.principalType === "User"
              ? t("A person")
              : event.principalType === "Agent"
                ? t("An agent")
                : t("The system")}
          </DescriptionItem>
          <DescriptionItem label={t("Agent")}>
            {event.agentName || event.agent?.name ? (
              <span>
                {event.agent?.name ?? event.agentName}
                {event.agentDefinitionVersion != null ? (
                  <span className="text-foreground-muted ml-1.5 text-xs tabular-nums">
                    {t("v{0}", event.agentDefinitionVersion)}
                  </span>
                ) : null}
              </span>
            ) : (
              <DescriptionEmpty />
            )}
          </DescriptionItem>
        </DescriptionList>
      </Section>

      <Section title={t("What")}>
        <DescriptionList layout="inline">
          <DescriptionItem label={t("Event")}>{auditKindLabel(t, event.kind)}</DescriptionItem>
          <DescriptionItem label={t("Tool")}>
            <Mono value={event.toolName} />
          </DescriptionItem>
          <DescriptionItem label={t("Effect")}>{orEmpty(event.toolEffect)}</DescriptionItem>
          <DescriptionItem label={t("Result")}>{orEmpty(event.resultSummary)}</DescriptionItem>
          <DescriptionItem label={t("Run")}>
            <Mono value={event.runId} />
          </DescriptionItem>
          <DescriptionItem label={t("Turn")}>
            <Mono value={event.turnId} />
          </DescriptionItem>
          <DescriptionItem label={t("Proposal")}>
            <Mono value={event.proposalId} />
          </DescriptionItem>
          {event.delegateCallId ? (
            <DescriptionItem label={t("Handed-off task")}>
              <Mono value={event.delegateCallId} />
            </DescriptionItem>
          ) : null}
        </DescriptionList>
      </Section>

      <Section title={t("Why")}>
        <DescriptionList layout="inline">
          <DescriptionItem label={t("Tier")}>
            {event.tier ? auditTierLabel(t, event.tier) : <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("Tier set by")}>
            {event.tierSource ? tierSourceLabel(t, event.tierSource) : <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("Held by")}>
            <HeldByChips heldBy={event.heldBy} />
          </DescriptionItem>
          <DescriptionItem label={t("Reason")}>{orEmpty(event.reason)}</DescriptionItem>
        </DescriptionList>
      </Section>

      <Section title={t("Changed")}>
        <div className="flex flex-col gap-3">
          <DescriptionList layout="inline">
            <DescriptionItem label={t("Record")}>
              {event.entityType && event.entityId ? (
                <span className="flex min-w-0 flex-col">
                  <span>{event.entityType}</span>
                  {recordHref ? (
                    <a
                      href={recordHref}
                      className="text-brand ui-focus-ring rounded-sm font-mono text-xs underline-offset-4 hover:underline"
                    >
                      {event.entityId}
                    </a>
                  ) : (
                    <Mono value={event.entityId} />
                  )}
                </span>
              ) : (
                <DescriptionEmpty />
              )}
            </DescriptionItem>
            <DescriptionItem label={t("Version before")} numeric>
              {orEmpty(event.versionBefore)}
            </DescriptionItem>
            <DescriptionItem label={t("Version after")} numeric>
              {orEmpty(event.versionAfter)}
            </DescriptionItem>
          </DescriptionList>
          {event.auditEntries.length > 0 ? (
            <div className="flex flex-col gap-1.5">
              <div className="flex items-center gap-2">
                <span className="text-foreground-subtle text-xs font-medium">
                  {t("Audit log entries")}
                </span>
                <Badge variant="neutral" appearance="outline">
                  {t("Matched by time")}
                </Badge>
              </div>
              <ul className="divide-border-subtle flex flex-col divide-y rounded-md border">
                {event.auditEntries.map((entry) => (
                  <li key={entry.id} className="flex flex-col gap-0.5 px-2.5 py-1.5 text-xs">
                    <span className="flex items-center justify-between gap-2">
                      <span>
                        {entry.operation} · {entry.resource}
                      </span>
                      <span className="text-foreground-muted tabular-nums">
                        {formatUnixDateTimeMedium(entry.timestamp)}
                      </span>
                    </span>
                    <span className="text-foreground-muted">
                      {entry.user?.name ?? entry.userId ?? t("An agent")}
                      {entry.comment ? ` · ${entry.comment}` : ""}
                    </span>
                  </li>
                ))}
              </ul>
              <p className="text-foreground-muted text-xs">
                {t(
                  "These are the audit log rows written for the same record, by the same principal, within this event's window. They are matched by time, not by a shared key.",
                )}
              </p>
            </div>
          ) : null}
        </div>
      </Section>

      <Section title={t("Provenance")}>
        <DescriptionList layout="inline">
          <DescriptionItem label={t("Read outside content")}>
            {yesNo(t, event.tainted)}
          </DescriptionItem>
          <DescriptionItem label={t("After outside content")}>
            {yesNo(t, event.externalContent)}
          </DescriptionItem>
          <DescriptionItem label={t("Where the write reaches")}>
            {orEmpty(event.egressClass)}
          </DescriptionItem>
          {event.taint ? (
            <DescriptionItem label={t("Outside content read")}>
              <JsonBlock value={event.taint} />
            </DescriptionItem>
          ) : null}
        </DescriptionList>
      </Section>

      {event.kind === "ModelCall" || event.model ? (
        <Section title={t("Model")}>
          <DescriptionList layout="inline">
            <DescriptionItem label={t("Provider")}>{orEmpty(event.providerKind)}</DescriptionItem>
            <DescriptionItem label={t("Model")}>
              <Mono value={event.model} />
            </DescriptionItem>
            <DescriptionItem label={t("Tokens in")} numeric>
              {formatTokens(event.inputTokens)}
            </DescriptionItem>
            <DescriptionItem label={t("Tokens out")} numeric>
              {formatTokens(event.outputTokens)}
            </DescriptionItem>
            <DescriptionItem label={t("Reasoning tokens")} numeric>
              {formatTokens(event.reasoningTokens)}
            </DescriptionItem>
            <DescriptionItem label={t("Cache read")} numeric>
              {formatTokens(event.cacheReadTokens)}
            </DescriptionItem>
            <DescriptionItem label={t("Cache written")} numeric>
              {formatTokens(event.cacheWriteTokens)}
            </DescriptionItem>
            <DescriptionItem label={t("Cost")} numeric>
              {orEmpty(formatUsd(event.costUsd))}
            </DescriptionItem>
            <DescriptionItem label={t("Attempt")} numeric>
              {orEmpty(event.attempt)}
            </DescriptionItem>
            <DescriptionItem label={t("Failover")}>{yesNo(t, event.failover)}</DescriptionItem>
            <DescriptionItem label={t("Latency")} numeric>
              {event.latencyMs != null ? formatLatency(event.latencyMs) : <DescriptionEmpty />}
            </DescriptionItem>
          </DescriptionList>
        </Section>
      ) : null}

      {event.arguments != null || event.redactedPaths.length > 0 ? (
        <Section title={t("Arguments")}>
          <div className="flex flex-col gap-2">
            {event.arguments != null ? <JsonBlock value={event.arguments} /> : null}
            <p className="text-foreground-muted text-xs">
              {t(
                "Values above what you may see on this record are shown as [withheld]. Confidential values were never recorded.",
              )}
            </p>
            {event.redactedPaths.length > 0 ? (
              <p className="text-foreground-muted text-xs">
                {t("Masked when recorded: {0}", event.redactedPaths.join(", "))}
              </p>
            ) : null}
            {event.argumentsTruncated ? (
              <Alert size="sm">
                <CircleAlertIcon />
                <AlertDescription>
                  {t("The arguments were longer than the trail keeps, so some values were cut.")}
                </AlertDescription>
              </Alert>
            ) : null}
          </div>
        </Section>
      ) : null}

      <Section title={t("Trace")}>
        <DescriptionList layout="inline">
          <DescriptionItem label={t("Trace")}>
            <TraceLink traceId={event.traceId} traceUrl={event.traceUrl} />
          </DescriptionItem>
          <DescriptionItem label={t("Trace id")}>
            <CopyableMono value={event.traceId} label={t("Copy trace id")} />
          </DescriptionItem>
          <DescriptionItem label={t("Span id")}>
            <Mono value={event.spanId} />
          </DescriptionItem>
          <DescriptionItem label={t("Call id")}>
            <Mono value={event.callId} />
          </DescriptionItem>
          <DescriptionItem label={t("Step key")}>
            <Mono value={event.stepKey} />
          </DescriptionItem>
        </DescriptionList>
      </Section>

      <Section title={t("Chain")}>
        <DescriptionList layout="inline">
          <DescriptionItem label={t("Seq")} numeric>
            #{event.seq.toLocaleString()}
          </DescriptionItem>
          <DescriptionItem label={t("Signed")}>
            {event.hashKeyId ? t("Yes, with key {0}", event.hashKeyId) : t("No")}
          </DescriptionItem>
          <DescriptionItem label={t("Previous hash")}>
            <CopyableMono value={event.prevHash} label={t("Copy hash")} />
          </DescriptionItem>
          <DescriptionItem label={t("Hash")}>
            <CopyableMono value={event.hash} label={t("Copy hash")} />
          </DescriptionItem>
          <DescriptionItem label={t("Recorded")}>
            {formatUnixDateTimeMedium(event.recordedAt)}
          </DescriptionItem>
          <DescriptionItem label={t("Source")}>
            <Mono value={event.sourceKey} />
          </DescriptionItem>
        </DescriptionList>
      </Section>
    </div>
  );
}

function JsonBlock({ value }: { value: unknown }) {
  return (
    <pre className="bg-sunken max-h-72 overflow-auto rounded-md border p-2 font-mono text-xs break-all whitespace-pre-wrap">
      {JSON.stringify(value, null, 2)}
    </pre>
  );
}
