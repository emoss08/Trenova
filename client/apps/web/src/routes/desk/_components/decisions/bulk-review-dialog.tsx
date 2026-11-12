import {
  visibilityLabel,
  channelLabel,
} from "@/components/assistant/proposal-preview/preview-format";
import { ProposalPreview } from "@/components/assistant/proposal-preview/proposal-preview";
import { ValueChange } from "@/components/assistant/proposal-preview/value-change";
import { presentProposal } from "@/components/assistant/proposal-presenters";
import type { PreviewMessage } from "@/lib/graphql/agent-preview";
import type { TemplatePart } from "@/lib/shared-template";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { useRef, useState } from "react";
import {
  batchReview,
  type BatchField,
  type BatchPreviewEntry,
  type BatchReview,
  type BatchRow,
} from "./batch-review-model";
import { flagOf } from "./decision-gates";
import { asAssistantProposal } from "./decision-presenters";
import type { PendingProposalNode } from "./use-pending-decisions";

/** The changes an approval of several would carry, and what each was read as. */
export type BulkReviewRequest = {
  nodes: PendingProposalNode[];
  entries: BatchPreviewEntry[];
  onConfirm: () => Promise<void>;
};

/**
 * The one look an approval of several changes takes: what they have alike
 * shown once (how a message goes out, the values every change sets alike, a
 * message's wording as a template with numbered places), then each change
 * with only what is its own, the ones with something to read first. What is
 * on this sheet is what the approval records.
 */
export function BulkReviewDialog({
  request,
  onClose,
}: {
  request: BulkReviewRequest | null;
  onClose: () => void;
}) {
  return (
    <Dialog open={request !== null} onOpenChange={(open) => !open && onClose()}>
      <DialogContent size="lg">
        {request && <Review request={request} onClose={onClose} />}
      </DialogContent>
    </Dialog>
  );
}

function Review({ request, onClose }: { request: BulkReviewRequest; onClose: () => void }) {
  const t = useT();
  const [pending, setPending] = useState(false);
  const sending = useRef(false);
  const review = batchReview(request.entries);
  const nodesById = new Map(request.nodes.map((node) => [node.id, node]));
  const count = request.nodes.length;
  const permanent = request.nodes.some(
    (node) => !presentProposal(asAssistantProposal(node)).reversible,
  );
  // The ones with something to read lead, so they are what is on screen.
  const rows = [...review.rows].sort(
    (a, b) => Number(flagged(b) !== null) - Number(flagged(a) !== null),
  );

  const confirm = async () => {
    if (sending.current) {
      return;
    }
    sending.current = true;
    setPending(true);
    try {
      await request.onConfirm();
      onClose();
    } finally {
      sending.current = false;
      setPending(false);
    }
  };

  return (
    <>
      <DialogHeader>
        <DialogTitle>
          {t("Approve {0, plural, one {# change} other {# changes}}?", count)}
        </DialogTitle>
        <DialogDescription>
          {permanent
            ? t("This is what your approval records. Some of these can't be undone once they run.")
            : t("This is what your approval records. Each can be undone after it runs.")}
        </DialogDescription>
      </DialogHeader>
      <ScrollArea className="-mx-1 max-h-[min(56vh,480px)]">
        <div className="flex flex-col gap-3.5 px-1">
          {review.layout === "compact" && <SharedPart review={review} />}
          <ul className="flex flex-col" aria-label={t("Changes waiting")}>
            {rows.map((row) => (
              <ReviewRow
                key={row.proposalId}
                row={row}
                layout={review.layout}
                fallback={titleOf(nodesById.get(row.proposalId))}
              />
            ))}
          </ul>
        </div>
      </ScrollArea>
      <DialogFooter>
        <Button variant="outline" onClick={onClose} disabled={pending}>
          {t("Cancel")}
        </Button>
        <Button onClick={() => void confirm()} isLoading={pending}>
          {t("Approve {0}", count)}
        </Button>
      </DialogFooter>
    </>
  );
}

function titleOf(node: PendingProposalNode | undefined): string {
  return node ? presentProposal(asAssistantProposal(node)).title : "";
}

function flagged(row: BatchRow) {
  return flagOf({ proposalId: row.proposalId, preview: row.preview, error: row.error });
}

function ReviewRow({
  row,
  layout,
  fallback,
}: {
  row: BatchRow;
  layout: BatchReview["layout"];
  fallback: string;
}) {
  const t = useT();
  const title = row.to.length > 0 ? row.to.join(", ") : row.title || fallback;

  return (
    <li className="flex flex-col gap-1.5 border-t border-border-subtle py-2.5 first:border-t-0">
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
        <b className="text-sm font-medium">{title}</b>
        {flagged(row) !== null && (
          <Badge variant="warning" dot className="ml-auto">
            {t("Something to look at")}
          </Badge>
        )}
      </div>
      {row.preview === null ? (
        <p className="text-sm text-foreground-muted">
          {row.error ?? t("This change could not be previewed.")}
        </p>
      ) : layout === "whole" ? (
        <ProposalPreview preview={row.preview} density="compact" />
      ) : (
        <>
          <Filled values={row.subjectValues} whole={row.subject} offset={0} />
          <Filled
            values={row.bodyValues}
            whole={row.body}
            offset={row.subjectValues?.length ?? 0}
          />
          <FieldList fields={row.fields} />
        </>
      )}
    </li>
  );
}

/** A numbered place in a template, and the same number beside what fills it on a row. */
function SlotMark({ slot }: { slot: number }) {
  return (
    <span className="mx-0.5 inline-flex min-w-4 items-center justify-center rounded bg-surface-active px-1 text-xs font-medium text-foreground-muted tabular-nums">
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

/** What every change of the review has alike, shown once above the rows. */
function SharedPart({ review }: { review: BatchReview }) {
  const t = useT();
  const { envelope, subject, body } = review;
  const hasTemplate = (subject !== null && subject.parts.length > 0) || body !== null;
  if (envelope === null && review.shared.length === 0 && !hasTemplate) {
    return null;
  }

  return (
    <section
      className="flex flex-col gap-2.5 rounded-lg bg-surface-hover px-3.5 py-3"
      aria-label={t("The same for every one")}
    >
      <h3 className="text-xs text-foreground-muted">{t("The same for every one")}</h3>
      {envelope !== null && <Envelope message={envelope} />}
      <FieldList fields={review.shared} />
      {subject !== null && subject.parts.length > 0 && (
        <p className="text-sm font-medium">
          <TemplateText parts={subject.parts} />
        </p>
      )}
      {body !== null && (
        <p className="text-sm leading-relaxed break-words whitespace-pre-wrap">
          <TemplateText parts={body.parts} />
        </p>
      )}
      {(subject?.slots ?? 0) + (body?.slots ?? 0) > 0 && (
        <p className="text-xs text-foreground-muted">
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
      <p className="max-h-40 overflow-y-auto rounded-md bg-surface-hover px-2.5 py-2 text-sm leading-relaxed break-words whitespace-pre-wrap">
        {whole}
      </p>
    );
  }
  if (values.length === 0) {
    return null;
  }

  return (
    <p className="flex flex-wrap items-baseline gap-x-3 gap-y-1 text-sm text-foreground-muted">
      {values.map((value, index) => (
        <span key={index} className="inline-flex items-baseline">
          <SlotMark slot={offset + index} />
          <span className="break-words">{value}</span>
        </span>
      ))}
    </p>
  );
}
