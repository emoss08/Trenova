import {
  channelLabel,
  visibilityLabel,
} from "@/components/assistant/proposal-preview/preview-format";
import {
  previewNeedsAttention,
  previewWarningText,
  previewWarningTone,
  visibleWarnings,
} from "@/components/assistant/proposal-preview/preview-warnings";
import {
  PreviewFailedNotice,
  PreviewSkeleton,
  ProposalPreview,
} from "@/components/assistant/proposal-preview/proposal-preview";
import { PREVIEW_STALE_TIME } from "@/components/assistant/proposal-preview/use-proposal-preview";
import { ValueChange } from "@/components/assistant/proposal-preview/value-change";
import { humanizeToolName } from "@/components/assistant/proposal-state";
import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import type { PreviewMessage } from "@/lib/graphql/agent-preview";
import { agentPreview } from "@/lib/queries/agent-preview";
import type { TemplatePart } from "@/lib/shared-template";
import { useQuery } from "@tanstack/react-query";
import { AlertCircleIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useState, type ReactNode } from "react";
import { QueueCard } from "../queue/queue";
import type { QueueLeave } from "../queue/use-queue-cursor";
import {
  approvableRows,
  batchReview,
  type BatchField,
  type BatchReview,
  type BatchRow,
} from "./batch-review-model";
import type { PendingProposalNode } from "./use-pending-decisions";

/**
 * What a batch decision carries, as the sheet tells the bar: the proposals an
 * approval takes, each with the digest of the preview the sheet showed, and
 * the ones a decline takes. Approval holds until every preview is read and
 * the person has scrolled the list to its end.
 */
export type BatchGate = {
  approveIds: string[];
  digests: { proposalId: string; digest: string }[];
  declineIds: string[];
  approvable: boolean;
  /** Why approval holds; null when it does not. */
  holding: "reading" | "unread" | "failed" | "empty" | null;
};

type BatchCardProps = {
  /** The proposals decided together, one tool's, in queue order. */
  nodes: PendingProposalNode[];
  leaving: QueueLeave | null;
  footer?: ReactNode;
  onGate: (gate: BatchGate) => void;
};

/** A row can be decided with the rest: its preview was read and its record has not moved. */
function batchable(row: BatchRow): boolean {
  return row.preview !== null && !row.preview.stale;
}

/** Rows that need a look lead, so what must be weighed is on screen first. */
function ordered(review: BatchReview): BatchRow[] {
  const flagged = (row: BatchRow) =>
    row.preview === null || previewNeedsAttention(row.preview, { inPlan: false });

  return [...review.rows.filter(flagged), ...review.rows.filter((row) => !flagged(row))];
}

function gateOf(
  nodes: readonly PendingProposalNode[],
  review: BatchReview | null,
  failed: boolean,
  excluded: ReadonlySet<string>,
  seenEnd: boolean,
): BatchGate {
  if (review !== null) {
    const digests = approvableRows(review, excluded);
    const approveIds = digests.map((row) => row.proposalId);

    return {
      approveIds,
      digests,
      declineIds: review.rows
        .filter((row) => batchable(row) && !excluded.has(row.proposalId))
        .map((row) => row.proposalId),
      approvable: approveIds.length > 0 && seenEnd,
      holding: approveIds.length === 0 ? "empty" : seenEnd ? null : "unread",
    };
  }

  const included = nodes.map((node) => node.id).filter((id) => !excluded.has(id));
  // Without the previews, a batch of internal changes is approved as not
  // reviewed, the way one card is; one that must be reviewed waits for them.
  const unreviewable = failed && !nodes.some((node) => node.reviewRequired);

  return {
    approveIds: unreviewable ? included : [],
    digests: [],
    declineIds: included,
    approvable: unreviewable && included.length > 0,
    holding: unreviewable ? null : failed ? "failed" : "reading",
  };
}

/**
 * Several changes of one kind decided together, on one sheet. What every
 * change has alike is shown once: how its message goes out, the values it
 * sets alike, and its wording as a template whose places are numbered. Each
 * row then shows what is its own: who it is for, what fills each place and
 * the values only it sets, with a way to open its whole preview. Changes that
 * differ in shape are each shown whole. A row whose record moved, or whose
 * preview could not be read, is decided on its own card.
 */
export function BatchCard({ nodes, leaving, footer, onGate }: BatchCardProps) {
  const t = useT();
  const ids = nodes.map((node) => node.id);
  const query = useQuery({ ...agentPreview.batch(ids), staleTime: PREVIEW_STALE_TIME });
  const review = query.data ? batchReview(query.data) : null;
  const [excluded, setExcluded] = useState<ReadonlySet<string>>(new Set());
  const [last, setLast] = useState<HTMLElement | null>(null);
  const [seenEnd, setSeenEnd] = useState(false);

  // The last row coming into view is the person having read to the end. A
  // browser that cannot say counts the list as read, as one card would be.
  const canObserve = typeof IntersectionObserver !== "undefined";
  const readToEnd = seenEnd || !canObserve;
  useEffect(() => {
    if (last === null || seenEnd || !canObserve) return;
    const observer = new IntersectionObserver((entries) => {
      if (entries.some((entry) => entry.isIntersecting)) {
        setSeenEnd(true);
      }
    });
    observer.observe(last);
    return () => observer.disconnect();
  }, [canObserve, last, seenEnd]);

  const gate = gateOf(nodes, review, query.isError, excluded, readToEnd);
  const gateKey = JSON.stringify(gate);
  useEffect(() => {
    onGate(JSON.parse(gateKey) as BatchGate);
  }, [gateKey, onGate]);

  const toggle = (id: string, included: boolean) =>
    setExcluded((previous) => {
      const next = new Set(previous);
      if (included) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  const rows = review ? ordered(review) : [];

  return (
    <QueueCard leaving={leaving}>
      <BatchWho nodes={nodes} />
      <h1>
        {humanizeToolName(nodes[0]?.toolName ?? "")}
        <span> · {t("{0, plural, one {# change} other {# changes}}", nodes.length)}</span>
      </h1>
      <p className="dk-dc2-why">
        {t("Everything each one would do is below. Untick any you want to leave in the queue.")}
      </p>

      {query.isPending ? (
        <PreviewSkeleton />
      ) : query.isError || review === null ? (
        <PreviewFailedNotice
          onRetry={() => void query.refetch()}
          retrying={query.isFetching}
          reviewRequired={nodes.some((node) => node.reviewRequired)}
        />
      ) : (
        <>
          {review.layout === "compact" && <SharedPart review={review} />}
          <ul className="mt-4 flex flex-col" aria-label={t("Changes in this decision")}>
            {rows.map((row, index) => (
              <BatchRowItem
                key={row.proposalId}
                ref={index === rows.length - 1 ? setLast : undefined}
                row={row}
                layout={review.layout}
                included={!excluded.has(row.proposalId)}
                onToggle={(included) => toggle(row.proposalId, included)}
              />
            ))}
          </ul>
        </>
      )}
      {footer}
    </QueueCard>
  );
}

function BatchWho({ nodes }: { nodes: readonly PendingProposalNode[] }) {
  const t = useT();
  const rt = useRichT();
  const agents = new Map<string, NonNullable<PendingProposalNode["run"]>["definition"]>();
  for (const node of nodes) {
    const definition = node.run?.definition ?? null;
    agents.set(definition?.id ?? node.run?.agentDefinitionId ?? "", definition);
  }
  const [first] = agents.values();

  return (
    <div className="dk-dc2-who">
      <DeskAgentTile agent={first ?? null} size="sm" />
      <span>
        {agents.size > 1
          ? rt(
              "<b>{0, plural, one {# agent} other {# agents}}</b> want to:",
              { b: (c) => <b>{c}</b> },
              agents.size,
            )
          : rt("<b>{0}</b> wants to:", { b: (c) => <b>{c}</b> }, first?.name ?? t("An agent"))}
      </span>
    </div>
  );
}

/** A numbered place in a template, and the same number beside what fills it on a row. */
function SlotMark({ slot }: { slot: number }) {
  return (
    <span className="mx-0.5 inline-flex min-w-4 items-center justify-center rounded bg-dsk-hover px-1 text-xs font-medium text-dsk-fg2 tabular-nums">
      {slot + 1}
    </span>
  );
}

function TemplateText({ parts }: { parts: readonly TemplatePart[] }) {
  return (
    <>
      {parts.map((part, index) =>
        part.kind === "text" ? (
          <span key={index}>{part.text}</span>
        ) : (
          <SlotMark key={index} slot={part.slot} />
        ),
      )}
    </>
  );
}

function Envelope({ message }: { message: PreviewMessage }) {
  const t = useT();

  return (
    <DescriptionList layout="inline">
      <DescriptionItem label={t("Channel")}>{channelLabel(message.channel, t)}</DescriptionItem>
      {message.from !== "" && <DescriptionItem label={t("From")}>{message.from}</DescriptionItem>}
      {message.cc.length > 0 && (
        <DescriptionItem label={t("Cc")}>{message.cc.join(", ")}</DescriptionItem>
      )}
      {message.bcc.length > 0 && (
        <DescriptionItem label={t("Bcc")}>{message.bcc.join(", ")}</DescriptionItem>
      )}
      {message.visibility !== "" && (
        <DescriptionItem label={t("Seen by")}>
          {visibilityLabel(message.visibility, t)}
        </DescriptionItem>
      )}
      {message.cadence !== "" && (
        <DescriptionItem label={t("Repeats")}>{message.cadence}</DescriptionItem>
      )}
      {message.attachments.length > 0 && (
        <DescriptionItem label={t("Attachments")}>{message.attachments.join(", ")}</DescriptionItem>
      )}
    </DescriptionList>
  );
}

function FieldList({ fields }: { fields: readonly BatchField[] }) {
  if (fields.length === 0) {
    return null;
  }

  return (
    <DescriptionList layout="inline">
      {fields.map(({ key, field, operation }) => (
        <DescriptionItem key={key} label={field.label}>
          <ValueChange change={field} operation={operation} />
        </DescriptionItem>
      ))}
    </DescriptionList>
  );
}

/** What every change in the batch has alike, shown once above the rows. */
function SharedPart({ review }: { review: BatchReview }) {
  const t = useT();
  const { envelope, subject, body } = review;
  const hasTemplate = (subject !== null && subject.parts.length > 0) || body !== null;
  if (envelope === null && review.shared.length === 0 && !hasTemplate) {
    return null;
  }

  return (
    <section
      className="mt-4 flex flex-col gap-2.5 rounded-lg bg-dsk-sunken px-3.5 py-3"
      aria-label={t("The same for every one")}
    >
      <h2 className="text-dsk-sm font-medium text-dsk-fg">{t("The same for every one")}</h2>
      {envelope !== null && <Envelope message={envelope} />}
      <FieldList fields={review.shared} />
      {subject !== null && subject.parts.length > 0 && (
        <p className="text-sm font-medium text-dsk-fg">
          <TemplateText parts={subject.parts} />
        </p>
      )}
      {body !== null && (
        <p className="text-sm leading-relaxed break-words whitespace-pre-wrap text-dsk-fg2">
          <TemplateText parts={body.parts} />
        </p>
      )}
      {(subject?.slots ?? 0) + (body?.slots ?? 0) > 0 && (
        <p className="text-xs text-dsk-subtle">
          {t("Each numbered place is filled in from the change's own row below.")}
        </p>
      )}
    </section>
  );
}

/** What fills each numbered place on one row, or the text whole when it has none. */
function Filled({
  values,
  whole,
  offset,
}: {
  values: string[] | null;
  whole: string;
  offset: number;
}) {
  if (values === null) {
    return whole === "" ? null : (
      <p className="max-h-40 overflow-y-auto rounded-md bg-dsk-sunken px-2.5 py-2 text-sm leading-relaxed break-words whitespace-pre-wrap text-dsk-fg2">
        {whole}
      </p>
    );
  }
  if (values.length === 0) {
    return null;
  }

  return (
    <p className="flex flex-wrap items-baseline gap-x-3 gap-y-1 text-sm text-dsk-fg2">
      {values.map((value, index) => (
        <span key={index} className="inline-flex items-baseline">
          <SlotMark slot={offset + index} />
          <span className="break-words">{value}</span>
        </span>
      ))}
    </p>
  );
}

function RowBadges({ row }: { row: BatchRow }) {
  const t = useT();
  if (row.preview === null) {
    return null;
  }
  if (row.preview.stale) {
    return (
      <Badge variant="warning" dot>
        {t("Changed since")}
      </Badge>
    );
  }

  return (
    <>
      {visibleWarnings(row.preview.warnings, { inPlan: false }).map((warning, index) => {
        const tone = previewWarningTone(warning);
        return (
          <Badge
            key={`${warning.code}-${index}`}
            variant={tone === "danger" ? "danger" : tone === "warning" ? "warning" : "info"}
            dot
          >
            {previewWarningText(warning, t)}
          </Badge>
        );
      })}
    </>
  );
}

function BatchRowItem({
  ref,
  row,
  layout,
  included,
  onToggle,
}: {
  ref?: (element: HTMLElement | null) => void;
  row: BatchRow;
  layout: BatchReview["layout"];
  included: boolean;
  onToggle: (included: boolean) => void;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const title = row.title || t("This change");
  const decidable = batchable(row);
  const subjectSlots = row.subjectValues?.length ?? 0;

  return (
    <li
      ref={ref}
      className={cn(
        "flex gap-3 border-t border-dsk-b-sub py-3 first:border-t-0",
        !included && decidable && "opacity-60",
      )}
    >
      {decidable && (
        <Checkbox
          className="mt-0.5"
          checked={included}
          onCheckedChange={(checked) => onToggle(checked === true)}
          aria-label={t("Include {0}", title)}
        />
      )}
      <div className={cn("flex min-w-0 flex-1 flex-col gap-1.5", !decidable && "pl-7")}>
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
          <b className="text-sm font-medium text-dsk-fg">{title}</b>
          {row.to.length > 0 && row.to.join(", ") !== title && (
            <span className="text-sm text-dsk-subtle">{t("to {0}", row.to.join(", "))}</span>
          )}
          <RowBadges row={row} />
        </div>

        {row.preview === null ? (
          <p className="flex flex-wrap items-center gap-x-1.5 text-sm text-dsk-error">
            <AlertCircleIcon className="size-3.5 shrink-0" aria-hidden />
            <span>{row.error ?? t("This change could not be previewed.")}</span>
            <span>{t("Decide it on its own card.")}</span>
          </p>
        ) : row.preview.stale ? (
          <p className="text-sm text-dsk-subtle">
            {t("Its record changed since it was proposed. Decide it on its own card.")}
          </p>
        ) : layout === "whole" ? (
          <ProposalPreview preview={row.preview} density="compact" />
        ) : (
          <>
            <Filled values={row.subjectValues} whole={row.subject} offset={0} />
            <Filled values={row.bodyValues} whole={row.body} offset={subjectSlots} />
            <FieldList fields={row.fields} />
          </>
        )}

        {row.preview !== null && layout === "compact" && (
          <>
            <Button
              variant="bare"
              size="bare"
              className="w-fit text-xs text-dsk-subtle hover:text-dsk-fg"
              aria-expanded={open}
              onClick={() => setOpen((current) => !current)}
            >
              {open ? t("Hide the whole change") : t("Show the whole change")}
            </Button>
            {open && (
              <div className="rounded-lg border border-dsk-b-sub px-3 py-2.5">
                <ProposalPreview preview={row.preview} density="compact" attention={false} />
              </div>
            )}
          </>
        )}
      </div>
    </li>
  );
}
