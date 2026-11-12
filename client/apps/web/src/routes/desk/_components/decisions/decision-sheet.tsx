import type { EditorFocus } from "@/components/assistant/proposal-editor";
import {
  channelLabel,
  visibilityLabel,
} from "@/components/assistant/proposal-preview/preview-format";
import { isPreviewConflict } from "@/components/assistant/proposal-preview/preview-gate";
import {
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
import { presentProposal } from "@/components/assistant/proposal-presenters";
import { humanizeToolName } from "@/components/assistant/proposal-state";
import { approvalFacts, recordCount } from "@/components/desk-chat/conversation/approval-facts";
import type { PreviewMessage } from "@/lib/graphql/agent-preview";
import { agentPreview } from "@/lib/queries/agent-preview";
import type { TemplatePart } from "@/lib/shared-template";
import { useQuery } from "@tanstack/react-query";
import { AlertCircleIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useRef, useState } from "react";
import { QueueCard } from "../queue/queue";
import { QueueKey } from "../queue/queue-bar";
import type { QueueLeave } from "../queue/use-queue-cursor";
import {
  batchReview,
  type BatchField,
  type BatchPreviewEntry,
  type BatchReview,
  type BatchRow,
} from "./batch-review-model";
import { ProposalDetail, type CardGate } from "./decision-card";
import { asAssistantProposal } from "./decision-presenters";
import type { PendingProposalNode } from "./use-pending-decisions";

/** What the ticked changes of a sheet come to, as the bar reads it. */
export type TickedGate = {
  /** Ticked changes an approval carries: read, current and on screen at least once. */
  approveIds: string[];
  /** The digest of each preview the sheet showed, for those it read. */
  digests: { proposalId: string; digest: string }[];
  /** Ticked changes a decline carries: every one that is not held. */
  declineIds: string[];
  approvable: boolean;
  /** Why approving the ticked changes holds; null when it does not. */
  holding: "reading" | "unseen" | "failed" | "none" | null;
};

/**
 * What the sheet tells the bar: the open change, decided alone while
 * nothing is ticked, and the ticked changes, decided together.
 */
export type SheetGate = {
  current: CardGate | null;
  ticked: TickedGate;
};

type DecisionSheetProps = {
  /** One kind of change, in queue order, at most MAX_BATCH of them, held ones included. */
  nodes: PendingProposalNode[];
  /** How many of this kind wait in all, when the sheet shows only part of them. */
  total: number;
  currentId: string;
  ticked: ReadonlySet<string>;
  onTick: (ids: readonly string[], ticked: boolean) => void;
  onSelect: (id: string) => void;
  leaving: QueueLeave | null;
  leavingIds: ReadonlySet<string>;
  onGate: (gate: SheetGate) => void;
  /** Turns the open change down with a reason already written. */
  onDeclineWith: (reason: string) => void;
  /** Opens the editor on the open change; absent when its values can't be changed. */
  onModify?: (focus?: EditorFocus) => void;
};

function canDecide(node: PendingProposalNode): boolean {
  return node.hold === null || node.hold === undefined;
}

/** Whether a change can be approved without its digest: an internal change whose preview failed. */
function unreviewedApprovable(
  node: PendingProposalNode,
  entry: BatchPreviewEntry | undefined,
  failed: boolean,
): boolean {
  return (failed || (entry !== undefined && entry.preview === null)) && !node.reviewRequired;
}

function currentGateOf(
  node: PendingProposalNode | undefined,
  entry: BatchPreviewEntry | undefined,
  failed: boolean,
  refetch: () => void,
  t: TranslateFn,
): CardGate | null {
  if (node === undefined || !canDecide(node)) {
    return null;
  }
  const preview = entry?.preview ?? null;
  const facts = approvalFacts(preview ?? undefined, t);
  const handleError = (error: unknown) => {
    if (!isPreviewConflict(error)) {
      return false;
    }
    refetch();
    return true;
  };

  return {
    approvable: preview !== null ? !preview.stale : unreviewedApprovable(node, entry, failed),
    digest: preview?.digest,
    refused: facts.refused.length,
    count: facts.count,
    handleError,
  };
}

function tickedGateOf(
  nodes: readonly PendingProposalNode[],
  entries: ReadonlyMap<string, BatchPreviewEntry>,
  pending: boolean,
  failed: boolean,
  ticked: ReadonlySet<string>,
  seen: (id: string) => boolean,
): TickedGate {
  const chosen = nodes.filter((node) => ticked.has(node.id) && canDecide(node));
  const approveIds: string[] = [];
  const digests: { proposalId: string; digest: string }[] = [];
  let unseen = 0;
  let unreadable = 0;
  for (const node of chosen) {
    const entry = entries.get(node.id);
    const preview = entry?.preview ?? null;
    if (preview !== null && !preview.stale) {
      if (!seen(node.id)) {
        unseen++;
        continue;
      }
      approveIds.push(node.id);
      digests.push({ proposalId: node.id, digest: preview.digest });
    } else if (unreviewedApprovable(node, entry, failed)) {
      approveIds.push(node.id);
    } else if (preview === null) {
      unreadable++;
    }
  }

  const holding =
    chosen.length === 0
      ? null
      : pending
        ? "reading"
        : unseen > 0
          ? "unseen"
          : approveIds.length === 0
            ? unreadable > 0
              ? "failed"
              : "none"
            : null;

  return {
    approveIds,
    digests,
    declineIds: chosen.map((node) => node.id),
    approvable: !pending && unseen === 0 && approveIds.length > 0,
    holding,
  };
}

/**
 * The changes of one kind waiting on a person, on one sheet. A change on its
 * own reads as it always has; several read as one: what they have alike is
 * shown once (how a message goes out, the values every change sets alike,
 * a message's wording as a template with numbered places), and each row
 * shows what is its own. The open row is drawn whole, with who asked, why,
 * the dry run and the ways forward. Rows can be ticked to decide together;
 * a ticked row counts as reviewed once it has been on screen.
 */
export function DecisionSheet({
  nodes,
  total,
  currentId,
  ticked,
  onTick,
  onSelect,
  leaving,
  leavingIds,
  onGate,
  onDeclineWith,
  onModify,
}: DecisionSheetProps) {
  const t = useT();
  const ids = nodes.map((node) => node.id);
  const query = useQuery({ ...agentPreview.batch(ids), staleTime: PREVIEW_STALE_TIME });
  const review = query.data ? batchReview(query.data) : null;
  const entries = new Map((query.data ?? []).map((entry) => [entry.proposalId, entry]));
  const rowsById = new Map((review?.rows ?? []).map((row) => [row.proposalId, row]));
  const current = nodes.find((node) => node.id === currentId);

  // A row counts as read once it has been in view. A browser that cannot
  // say counts every row as read, as it would one card.
  const canObserve = typeof IntersectionObserver !== "undefined";
  const [seenIds, setSeenIds] = useState<ReadonlySet<string>>(new Set());
  const elements = useRef(new Map<string, HTMLElement>());
  const observer = useRef<IntersectionObserver | null>(null);
  useEffect(() => {
    if (!canObserve) return;
    const io = new IntersectionObserver((observed) => {
      const visible = observed
        .filter((entry) => entry.isIntersecting)
        .map((entry) => (entry.target as HTMLElement).dataset.proposalId ?? "")
        .filter((id) => id !== "");
      if (visible.length === 0) return;
      setSeenIds((previous) =>
        visible.every((id) => previous.has(id)) ? previous : new Set([...previous, ...visible]),
      );
    });
    observer.current = io;
    for (const element of elements.current.values()) {
      io.observe(element);
    }
    return () => {
      io.disconnect();
      observer.current = null;
    };
  }, [canObserve]);
  const track = (id: string) => (element: HTMLElement | null) => {
    if (element === null) return;
    elements.current.set(id, element);
    observer.current?.observe(element);
    return () => {
      elements.current.delete(id);
      observer.current?.unobserve(element);
    };
  };
  const seen = (id: string) => !canObserve || id === currentId || seenIds.has(id);

  const gate: SheetGate = {
    current: currentGateOf(
      current,
      entries.get(currentId),
      query.isError,
      () => void query.refetch(),
      t,
    ),
    ticked: tickedGateOf(nodes, entries, query.isPending, query.isError, ticked, seen),
  };
  // The bar hears of a change in what the sheet would decide, not of every render.
  const gateKey = JSON.stringify(gate);
  const reportGate = useLatestCallback(() => onGate(gate));
  useEffect(() => {
    reportGate();
  }, [gateKey, reportGate]);

  const decidable = nodes.filter(canDecide);
  const several = nodes.length > 1;
  const allLeaving = nodes.every((node) => leavingIds.has(node.id));

  return (
    <QueueCard leaving={allLeaving ? leaving : null}>
      <SheetHeading nodes={nodes} total={total} current={current} review={review} />
      {several && decidable.length > 1 && (
        <Selection
          decidable={decidable}
          ticked={ticked}
          holding={gate.ticked.holding}
          onTick={onTick}
        />
      )}

      {query.isError && (
        <PreviewFailedNotice
          onRetry={() => void query.refetch()}
          retrying={query.isFetching}
          reviewRequired={nodes.some((node) => node.reviewRequired)}
        />
      )}
      {review !== null && review.layout === "compact" && several && <SharedPart review={review} />}

      <ul className={cn(several && "mt-4", "flex flex-col")} aria-label={t("Changes waiting")}>
        {nodes.map((node) => (
          <SheetRow
            key={node.id}
            node={node}
            row={rowsById.get(node.id) ?? null}
            layout={review?.layout ?? "whole"}
            reading={query.isPending}
            several={several}
            open={node.id === currentId}
            ticked={ticked.has(node.id)}
            leaving={leavingIds.has(node.id) ? leaving : null}
            track={track(node.id)}
            onTick={(value) => onTick([node.id], value)}
            onSelect={() => onSelect(node.id)}
            onDeclineWith={onDeclineWith}
            onModify={onModify}
            canChange={gate.current?.approvable ?? false}
          />
        ))}
      </ul>
    </QueueCard>
  );
}

function SheetHeading({
  nodes,
  total,
  current,
  review,
}: {
  nodes: readonly PendingProposalNode[];
  total: number;
  current: PendingProposalNode | undefined;
  review: BatchReview | null;
}) {
  const t = useT();
  if (nodes.length === 1 && current !== undefined) {
    const view = presentProposal(asAssistantProposal(current));
    const preview = review?.rows[0]?.preview ?? undefined;
    const facts = approvalFacts(preview, t);

    return (
      <h1>
        {view.title}
        {facts.count > 0 && <span> · {recordCount(facts.resource, facts.count, t)}</span>}
      </h1>
    );
  }

  return (
    <h1>
      {humanizeToolName(nodes[0]?.toolName ?? "")}
      <span>
        {" · "}
        {total > nodes.length
          ? t("the first {0} of {1}", nodes.length, total)
          : t("{0, plural, one {# change} other {# changes}}", nodes.length)}
      </span>
    </h1>
  );
}

/** Ticking every change of the sheet, or none, and what holds approving the ticked ones. */
function Selection({
  decidable,
  ticked,
  holding,
  onTick,
}: {
  decidable: readonly PendingProposalNode[];
  ticked: ReadonlySet<string>;
  holding: TickedGate["holding"];
  onTick: (ids: readonly string[], ticked: boolean) => void;
}) {
  const t = useT();
  const count = decidable.filter((node) => ticked.has(node.id)).length;
  const all = count === decidable.length;
  const ids = decidable.map((node) => node.id);

  return (
    <div className="dk-dc2-like">
      <Checkbox
        checked={all}
        indeterminate={count > 0 && !all}
        onCheckedChange={(checked) => onTick(ids, checked === true)}
        aria-label={t("Select all {0}", decidable.length)}
      />
      {count === 0 ? (
        <span>{t("Tick changes to decide them together")}</span>
      ) : (
        <>
          <b>{t("{0, plural, one {# selected} other {# selected}}", count)}</b>
          {holding === "reading" && <span>{t("Reading every change…")}</span>}
          {holding === "unseen" && (
            <span>{t("Scroll past each selected change to approve them")}</span>
          )}
          {holding === "failed" && <span>{t("The changes could not be read")}</span>}
          {holding === "none" && <span>{t("None of these can be approved together")}</span>}
          <Button
            variant="bare"
            size="bare"
            className="ml-auto text-sm font-medium text-dsk-fg underline decoration-dsk-fg/30 underline-offset-2 hover:decoration-current"
            onClick={() => onTick(ids, false)}
          >
            {t("Clear")}
          </Button>
        </>
      )}
      {count === 0 && <QueueKey>X</QueueKey>}
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

/** What every change on the sheet has alike, shown once above the rows. */
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

function RowBadges({ node, row }: { node: PendingProposalNode; row: BatchRow | null }) {
  const t = useT();
  if (!canDecide(node)) {
    return <Badge dot>{t("Held")}</Badge>;
  }
  const preview = row?.preview ?? null;
  if (preview === null) {
    return null;
  }
  if (preview.stale) {
    return (
      <Badge variant="warning" dot>
        {t("Changed since")}
      </Badge>
    );
  }

  return (
    <>
      {visibleWarnings(preview.warnings, { inPlan: false }).map((warning, index) => {
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

/** A closed row: what it has of its own, or its preview whole when the sheet's changes differ. */
function RowSummary({ row, layout }: { row: BatchRow | null; layout: BatchReview["layout"] }) {
  const t = useT();
  if (row === null) {
    return null;
  }
  if (row.preview === null) {
    return (
      <p className="flex flex-wrap items-center gap-x-1.5 text-sm text-dsk-error">
        <AlertCircleIcon className="size-3.5 shrink-0" aria-hidden />
        <span>{row.error ?? t("This change could not be previewed.")}</span>
      </p>
    );
  }
  if (row.preview.stale) {
    return (
      <p className="text-sm text-dsk-subtle">
        {t("Its record changed since it was proposed. It can only be declined.")}
      </p>
    );
  }
  if (layout === "whole") {
    return <ProposalPreview preview={row.preview} density="compact" />;
  }

  return (
    <>
      <Filled values={row.subjectValues} whole={row.subject} offset={0} />
      <Filled values={row.bodyValues} whole={row.body} offset={row.subjectValues?.length ?? 0} />
      <FieldList fields={row.fields} />
    </>
  );
}

function SheetRow({
  node,
  row,
  layout,
  reading,
  several,
  open,
  ticked,
  leaving,
  track,
  onTick,
  onSelect,
  onDeclineWith,
  onModify,
  canChange,
}: {
  node: PendingProposalNode;
  row: BatchRow | null;
  layout: BatchReview["layout"];
  reading: boolean;
  several: boolean;
  open: boolean;
  ticked: boolean;
  leaving: QueueLeave | null;
  track: (element: HTMLElement | null) => (() => void) | undefined;
  onTick: (ticked: boolean) => void;
  onSelect: () => void;
  onDeclineWith: (reason: string) => void;
  onModify?: (focus?: EditorFocus) => void;
  canChange: boolean;
}) {
  const t = useT();
  const view = presentProposal(asAssistantProposal(node));
  const title = row?.title || view.title || t("This change");
  const decidable = canDecide(node);
  const element = useRef<HTMLLIElement | null>(null);

  // The open row is brought into view, whichever way it was opened.
  useEffect(() => {
    if (open && several) {
      element.current?.scrollIntoView?.({ block: "nearest" });
    }
  }, [open, several]);

  const ref = (li: HTMLLIElement | null) => {
    element.current = li;
    return track(li);
  };

  return (
    <li
      ref={ref}
      data-proposal-id={node.id}
      aria-current={open ? "true" : undefined}
      className={cn(
        "flex gap-3 transition-[opacity,transform] duration-300",
        several && "border-t border-dsk-b-sub py-3 first:border-t-0",
        open && several && "rounded-lg bg-dsk-hover/40 px-2",
        leaving === "ok" && "translate-x-4 opacity-0",
        leaving === "no" && "-translate-x-4 opacity-0",
      )}
    >
      {several && decidable && (
        <Checkbox
          className="mt-0.5"
          checked={ticked}
          onCheckedChange={(checked) => onTick(checked === true)}
          aria-label={t("Select {0}", title)}
        />
      )}
      <div className={cn("flex min-w-0 flex-1 flex-col gap-1.5", several && !decidable && "pl-7")}>
        {several && (
          <Button
            variant="bare"
            size="bare"
            className="flex w-fit flex-wrap items-center gap-x-2 gap-y-1 text-left"
            aria-expanded={open}
            onClick={onSelect}
          >
            <b className="text-sm font-medium text-dsk-fg">{title}</b>
            {row !== null && row.to.length > 0 && row.to.join(", ") !== title && (
              <span className="text-sm font-normal text-dsk-subtle">
                {t("to {0}", row.to.join(", "))}
              </span>
            )}
            <RowBadges node={node} row={row} />
          </Button>
        )}

        {open ? (
          reading ? (
            <>
              <ProposalDetail
                node={node}
                preview={null}
                onDeclineWith={onDeclineWith}
                onModify={onModify}
                canChange={false}
              />
              <PreviewSkeleton />
            </>
          ) : (
            <ProposalDetail
              node={node}
              preview={row?.preview ?? null}
              onDeclineWith={onDeclineWith}
              onModify={onModify}
              canChange={canChange}
            />
          )
        ) : (
          <RowSummary row={row} layout={layout} />
        )}
      </div>
    </li>
  );
}
