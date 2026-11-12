import type { EditorFocus } from "@/components/assistant/proposal-editor";
import {
  channelLabel,
  visibilityLabel,
} from "@/components/assistant/proposal-preview/preview-format";
import { editableParam } from "@/components/assistant/proposal-preview/preview-warnings";
import {
  PreviewSkeleton,
  ProposalPreview,
} from "@/components/assistant/proposal-preview/proposal-preview";
import { presentProposal } from "@/components/assistant/proposal-presenters";
import { approvalFacts, OUTCOME_FIELD } from "@/components/desk-chat/conversation/approval-facts";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import type {
  PreviewMessage,
  ProposalPreview as ProposalPreviewData,
} from "@/lib/graphql/agent-preview";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState, type ReactNode } from "react";
import { QueueCard } from "../queue/queue";
import type { QueueLeave } from "../queue/use-queue-cursor";
import type { BatchPreviewEntry } from "./batch-review-model";
import { diffRows, dryRun, Highlights, Who, type DiffRow } from "./decision-card";
import { refusalsOf } from "./decision-gates";
import { DecisionNotices } from "./decision-notices";
import { asAssistantProposal } from "./decision-presenters";
import { rowSubject } from "./decision-row";
import type { PendingProposalNode } from "./use-pending-decisions";

/** The most record rows the change lists before it says how many more there are. */
const SHOWN_ROWS = 8;
/** A reason longer than this is folded to its first lines. */
const LONG_REASON = 220;

/**
 * One change, read whole, and nothing else on the page: who asks and when,
 * what it is and what it is about, why, whether it can be taken back, what
 * to know before deciding with the fix beside it, and then the change itself
 * drawn as it will happen: a message as it goes out, one value across many
 * records as a line each, anything else record by record. Under it, one line
 * says what checking it against the records came to.
 */
export function DecisionChange({
  node,
  entry,
  reading,
  failed,
  onRetry,
  leaving,
  onModify,
  canChange,
}: {
  node: PendingProposalNode;
  entry: BatchPreviewEntry | undefined;
  /** The change's preview is still being read. */
  reading: boolean;
  /** The preview could not be read at all. */
  failed: boolean;
  onRetry: () => void;
  leaving: QueueLeave | null;
  /** Opens the editor, on one value when a refusal named it; absent when values can't be changed. */
  onModify?: (focus?: EditorFocus) => void;
  /** Whether the change can be approved as it stands, so a value it names can be edited. */
  canChange: boolean;
}) {
  const t = useT();
  const proposal = asAssistantProposal(node);
  const view = presentProposal(proposal);
  const preview = entry?.preview ?? null;
  const subject = rowSubject(entry, { title: "", summary: "" }, t).title;
  const blocked = refusalsOf(entry).length > 0;
  const editable = proposal.fields.length > 0 && onModify !== undefined;
  const run = preview !== null && !blocked && !preview.stale ? dryRun(preview, t) : null;

  return (
    <QueueCard leaving={leaving} className="flex flex-col gap-4.5">
      <div className="flex flex-col gap-2.5">
        <Who node={node} />
        <h1>
          {view.title}
          {subject !== "" && subject !== view.title && <span> · {subject}</span>}
        </h1>
        <Reason text={node.rationale || view.summary} />
        <div className="flex flex-wrap gap-1.5">
          <Badge variant={view.reversible ? "success" : "warning"} dot>
            {view.reversible ? t("Reversible") : t("Can't be undone")}
          </Badge>
        </div>
      </div>

      {preview !== null && (
        <DecisionNotices
          preview={preview}
          canChange={editable ? (param) => editableParam(param, proposal.fields) : undefined}
          onChange={
            editable && canChange
              ? (reason) => onModify?.({ param: reason.param, label: reason.label })
              : undefined
          }
        />
      )}

      {preview !== null ? (
        <Change preview={preview} />
      ) : reading ? (
        <Sheet>
          <div className="p-4">
            <PreviewSkeleton />
          </div>
        </Sheet>
      ) : (
        <>
          <Unread
            text={
              entry?.error ??
              (node.reviewRequired
                ? t(
                    "What this would change could not be loaded. It has to be reviewed before it can be approved, so try again.",
                  )
                : t(
                    "What this would change could not be loaded. Approving now is recorded as approved without a preview.",
                  ))
            }
            onRetry={failed ? onRetry : undefined}
          />
          <Highlights rows={view.highlights} />
        </>
      )}

      {run && (
        <p
          className={cn(
            "flex items-start gap-2 text-dsk-base",
            run.warn ? "text-dsk-warn-fg" : "text-dsk-subtle",
          )}
        >
          <span className={cn("mt-0.5 shrink-0", !run.warn && "text-dsk-success-fg")}>
            <DeskIcon name={run.warn ? "alert" : "shield"} size={13} stroke={2} />
          </span>
          <span>
            <span className="sr-only">{t("Dry run")}: </span>
            {run.text}
          </span>
        </p>
      )}
    </QueueCard>
  );
}

/** Why the agent asks. Agents can run long, so a long reason folds to its first lines. */
function Reason({ text }: { text: string }) {
  const t = useT();
  const [open, setOpen] = useState(false);
  if (text === "") {
    return null;
  }
  const long = text.length > LONG_REASON;

  return (
    <div className="flex flex-col items-start gap-1">
      <p
        className={cn(
          "max-w-[60ch] text-base leading-relaxed break-words text-dsk-fg2",
          long && !open && "line-clamp-3",
        )}
      >
        {text}
      </p>
      {long && (
        <Button
          variant="bare"
          size="bare"
          className="text-dsk-sm text-dsk-subtle hover:text-dsk-fg"
          aria-expanded={open}
          onClick={() => setOpen((current) => !current)}
        >
          {open ? t("Show less") : t("Show more")}
        </Button>
      )}
    </div>
  );
}

/** The raised sheet the change itself is drawn on. */
function Sheet({ caption, children }: { caption?: ReactNode; children: ReactNode }) {
  return (
    <div className="overflow-hidden rounded-xl bg-dsk-raised shadow-(--dsk-lift)">
      {caption && (
        <div className="flex items-center gap-2 border-b border-dsk-b-sub px-3.5 py-2.5 text-dsk-sm text-dsk-subtle">
          {caption}
        </div>
      )}
      {children}
    </div>
  );
}

function Unread({ text, onRetry }: { text: string; onRetry?: () => void }) {
  const t = useT();

  return (
    <div
      role="status"
      className="flex items-center gap-2.5 rounded-[10px] border border-warning-border bg-warning-subtle px-3 py-2.5 text-sm text-dsk-fg2"
    >
      <span aria-hidden className="size-1.5 shrink-0 rounded-full bg-warning" />
      <span className="min-w-0 flex-1 break-words">{text}</span>
      {onRetry && (
        <Button size="sm" variant="secondary" onClick={onRetry}>
          {t("Try again")}
        </Button>
      )}
    </div>
  );
}

function Change({ preview }: { preview: ProposalPreviewData }) {
  const t = useT();
  const shown = preview.changes.filter((change) => !change.withheld);
  const message = shown.length === 1 ? shown[0].message : null;
  if (message && preview.withheldCount === 0 && shown[0].fields.length === 0) {
    return <MessageSheet message={message} />;
  }
  const rows = diffRows(preview, t);
  if (rows !== null && rows.length > 0) {
    const label =
      shown[0]?.fields.find((field) => !field.withheld && field.path !== OUTCOME_FIELD)?.label ??
      "";
    return <DiffSheet label={label} rows={rows} count={approvalFacts(preview, t).count} />;
  }

  return (
    <Sheet>
      <div className="p-4">
        <ProposalPreview preview={preview} attention={false} />
      </div>
    </Sheet>
  );
}

function initials(name: string): string {
  return name
    .split(/\s+/u)
    .filter((part) => part !== "")
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join("");
}

/** A message as the person it goes to will read it: who, the subject, the words. */
function MessageSheet({ message }: { message: PreviewMessage }) {
  const t = useT();
  const extras =
    message.from !== "" ||
    message.cc.length > 0 ||
    message.bcc.length > 0 ||
    message.visibility !== "" ||
    message.cadence !== "" ||
    message.attachments.length > 0;

  return (
    <Sheet
      caption={
        <>
          <DeskIcon name="mail" size={13} />
          <b className="font-medium text-dsk-fg2">{channelLabel(message.channel, t)}</b>
        </>
      }
    >
      <div className="flex flex-col gap-2.5 px-4 pt-3.5 pb-4">
        <div className="flex items-center gap-2 text-dsk-sm text-dsk-subtle">
          {message.to.length === 1 && (
            <span
              aria-hidden
              className="grid size-5.5 shrink-0 place-items-center rounded-full bg-dsk-hover text-3xs font-semibold text-dsk-fg2"
            >
              {initials(message.to[0])}
            </span>
          )}
          <span className="min-w-0 break-words">
            {message.to.length > 0 ? t("to {0}", message.to.join(", ")) : t("No one to send it to")}
          </span>
        </div>
        {message.subject !== "" && (
          <p className="text-base font-semibold text-dsk-fg">{message.subject}</p>
        )}
        {message.body !== "" ? (
          <p className="text-base leading-relaxed break-words whitespace-pre-wrap text-dsk-fg2">
            {message.body}
          </p>
        ) : (
          <p className="text-dsk-sm text-dsk-subtle">{t("The message has no body.")}</p>
        )}
        {message.bodyTruncated && (
          <p className="text-xs text-dsk-subtle">
            {t("The message is longer than a preview keeps; the rest is not shown.")}
          </p>
        )}
      </div>
      {extras && (
        <div className="border-t border-dsk-b-sub px-4 py-3">
          <DescriptionList layout="inline">
            {message.from !== "" && (
              <DescriptionItem label={t("From")}>{message.from}</DescriptionItem>
            )}
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
              <DescriptionItem label={t("Attachments")}>
                {message.attachments.join(", ")}
              </DescriptionItem>
            )}
          </DescriptionList>
        </div>
      )}
    </Sheet>
  );
}

/** One value set across records, a line each: what it was, then what it becomes. */
function DiffSheet({ label, rows, count }: { label: string; rows: DiffRow[]; count: number }) {
  const t = useT();
  const shown = rows.slice(0, SHOWN_ROWS);

  return (
    <Sheet caption={<b className="font-medium text-dsk-fg2">{label}</b>}>
      <div className="flex flex-col">
        {shown.map((row, index) => (
          <div
            key={`${row.key}-${index}`}
            className="grid grid-cols-[minmax(0,1fr)_auto_auto_auto] items-center gap-x-3 border-t border-dsk-b-sub px-3.5 py-2.25 text-sm tabular-nums first:border-t-0"
          >
            <span className="truncate text-dsk-fg">{row.key}</span>
            <s className="text-dsk-subtle">{row.before}</s>
            <i aria-hidden className="text-dsk-faint not-italic">
              →
            </i>
            <b
              className={cn("font-medium", row.refused ? "text-danger-foreground" : "text-dsk-fg")}
              title={row.refused ?? undefined}
            >
              <span className="sr-only">{t("to")} </span>
              {row.refused ? t("Refused") : row.after}
              {row.refused && <span className="sr-only">: {row.refused}</span>}
            </b>
          </div>
        ))}
        {count > shown.length && (
          <div className="border-t border-dsk-b-sub px-3.5 py-2.25 text-dsk-sm text-dsk-subtle">
            {t("+ {0} more like these", count - shown.length)}
          </div>
        )}
      </div>
    </Sheet>
  );
}
