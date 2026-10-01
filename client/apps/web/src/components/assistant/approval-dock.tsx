import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatAltShortcut, formatShortcut } from "@trenova/shared/lib/shortcuts";
import { cn } from "@trenova/shared/lib/utils";
import type { Tone } from "@/components/kpi/tone";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { decideMyPlan, decideMyProposal, decideMyProposals } from "@/lib/graphql/agent-decisions";
import type { ProposalPreview as ProposalPreviewData } from "@/lib/graphql/agent-preview";
import {
  invalidateProposalViews,
  markPlanDecided,
  markProposalDecided,
} from "@/lib/proposal-cache";
import type { AssistantProposal, ProposalDecision } from "@/types/assistant";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  ChevronRightIcon,
  ClockIcon,
  PencilIcon,
  TriangleAlertIcon,
  type LucideIcon,
} from "lucide-react";
import { useCallback, useEffect, useId, useMemo, useRef, useState, type ReactNode } from "react";
import { toast } from "sonner";
import type { ApprovalEntry } from "./approval-queue";
import { ProposedBy } from "./decision-chrome";
import { useDecisionFollowUp } from "./decision-follow-up";
import { Highlights, StepList, previewsByStep } from "./decision-outcomes";
import { FloatingSlot } from "./floating-slot";
import { ProposalEditor, type EditorFocus, type ProposalEditorRequest } from "./proposal-editor";
import { changedValues, draftFromArguments, validateDraft } from "./proposal-edits";
import { previewFigure, previewSubject } from "./proposal-preview/preview-format";
import { batchPreviewDigests, canApprove, gateDigest } from "./proposal-preview/preview-gate";
import {
  askAgentMessage,
  askAgentPlanMessage,
  editableParam,
  previewNeedsAttention,
} from "./proposal-preview/preview-warnings";
import {
  PreviewAttention,
  PreviewLoadState,
  PreviewSkeleton,
  ProposalPreview,
  StaleNotice,
  type WouldFailActions,
} from "./proposal-preview/proposal-preview";
import {
  useApprovalGate,
  useDraftPreview,
  usePlanPreview,
  useProposalPreview,
} from "./proposal-preview/use-proposal-preview";
import { hasPresenter, presentProposal, type ProposalView } from "./proposal-presenters";
import { previewOutcomes, subsetCountLabel, subsetTally } from "./record-subset";
import { RecordSubsetField } from "./record-subset-field";

/** The most a note to the agent may hold; the server refuses more. */
export const MAX_DECISION_NOTE_LENGTH = 2000;

/** Past this many changes in one answer, each opens on demand instead of at once. */
export const BATCH_OPEN_LIMIT = 5;

/** The reason code a rejection of several proposals records; the server asks for one. */
const BATCH_REJECT_REASON = "rejected_in_conversation";

const NO_OUTCOMES: ReadonlyMap<string, string> = new Map();

export type ApprovalDockProps = {
  threadId: string;
  /** The decision on show. */
  entry: ApprovalEntry;
  /** Its place in the queue, from 1, and how many wait. */
  position: number;
  total: number;
  compact?: boolean;
  /** Whether the agent can still answer, so a rejection may carry words for it. */
  canTell: boolean;
  /** Puts off everything waiting; the box becomes a pill above the composer. */
  onDefer: () => void;
  /** Told once a decision on the entry was recorded. */
  onDecided?: (entry: ApprovalEntry) => void;
  /** Measured by the thread, as the composer is, so the last message is never hidden. */
  ref?: React.Ref<HTMLDivElement>;
};

/**
 * The approval box: a waiting decision in the composer's place.
 *
 * While the conversation has a proposal or plan waiting on the person, the
 * box stands where they would type, shows what the change would do, and
 * takes one answer for it: approve what is shown, reject it, tell the agent
 * what to do instead, or put the decision off. One decision at a time, the
 * oldest first; the next takes its place once this one is recorded. The
 * transcript keeps only a line for each, so the only buttons that decide are
 * here, and the conversation's Desk page and the floating panel share it.
 *
 * Keys: ⌘/Ctrl+Enter approves, Esc opens or closes the note to the agent,
 * Alt+L decides later. Enter alone never approves.
 */
export function ApprovalDock({ ref, compact = false, ...props }: ApprovalDockProps) {
  return (
    <FloatingSlot ref={ref} compact={compact}>
      <DockEntry key={props.entry.key} compact={compact} {...props} />
    </FloatingSlot>
  );
}

type DockEntryProps = Omit<ApprovalDockProps, "ref" | "compact"> & { compact: boolean };

function DockEntry(props: DockEntryProps) {
  switch (props.entry.kind) {
    case "plan":
      return <PlanDock {...props} entry={props.entry} />;
    case "batch":
      return <BatchDock {...props} entry={props.entry} />;
    default:
      return <ProposalDock {...props} entry={props.entry} />;
  }
}

/**
 * The note a rejection carries to the agent. Open, it is the box's focus:
 * ⌘/Ctrl+Enter or Enter sends it, Shift+Enter breaks the line, Esc puts it
 * away and keeps what was typed.
 */
type DockNote = {
  telling: boolean;
  note: string;
  setNote: (note: string) => void;
  open: (prefill?: string) => void;
  close: () => void;
};

function useDockNote(): DockNote {
  const [telling, setTelling] = useState(false);
  const [note, setNote] = useState("");

  const open = useCallback((prefill?: string) => {
    if (prefill !== undefined) {
      setNote(prefill.slice(0, MAX_DECISION_NOTE_LENGTH));
    }
    setTelling(true);
  }, []);
  const close = useCallback(() => setTelling(false), []);

  return { telling, note, setNote, open, close };
}

/** What one kind of decision hands the box to draw and act on. */
type DockFrameProps = {
  title: string;
  /** The record the decision is about, after the title. */
  subject?: string;
  /** The amount it comes to, when it moves money. */
  figure?: string | null;
  /** How many records or changes it covers. */
  count?: string;
  severity?: ProposalView["severity"];
  position: number;
  total: number;
  compact: boolean;
  permanent: boolean;
  byline?: ReactNode;
  /** The decision in a sentence, under the title. */
  summary: ReactNode;
  /** What the person has to see before answering; never folded away. */
  attention?: ReactNode;
  /** Every record it changes and why, behind "Details". */
  details: ReactNode;
  /** Keeps the details mounted while folded, for rows that report what they read. */
  keepDetailsMounted?: boolean;
  /** Dialogs the box opens; drawn in the box so its keys stay with it. */
  overlay?: ReactNode;
  approveLabel: string;
  rejectLabel: string;
  canApprove: boolean;
  /** Which answer is on its way to the server, if any. */
  pending: "approve" | "reject" | "tell" | null;
  onApprove: () => void;
  onReject: () => void;
  onTell: (note: string) => void;
  onModify?: () => void;
  modifyDisabled?: boolean;
  canTell: boolean;
  note: DockNote;
  onDefer: () => void;
};

const SEVERITY_BADGE: Record<
  Tone,
  "danger" | "warning" | "info" | "success" | "brand" | "neutral"
> = {
  danger: "danger",
  warning: "warning",
  info: "info",
  success: "success",
  brand: "brand",
  muted: "neutral",
};

/**
 * The box's chrome and its keys, the same for a proposal, a plan or several
 * changes decided together. It reads like a permission prompt: one line for
 * what would happen and what it comes to, one sentence for the rest, the
 * answers in one row, and every record it touches folded behind "Details".
 * What would stop the write or has moved under it stays in view.
 */
function DockFrame({
  title,
  subject = "",
  figure = null,
  count,
  severity = null,
  position,
  total,
  compact,
  permanent,
  byline,
  summary,
  attention,
  details,
  keepDetailsMounted = false,
  overlay,
  approveLabel,
  rejectLabel,
  canApprove: approvable,
  pending,
  onApprove,
  onReject,
  onTell,
  onModify,
  modifyDisabled = false,
  canTell,
  note,
  onDefer,
}: DockFrameProps) {
  const t = useT();
  const titleId = useId();
  const noteId = useId();
  const sectionRef = useRef<HTMLElement>(null);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const busy = pending !== null;
  const trimmed = note.note.trim();
  const canSend = trimmed !== "" && !busy;

  const sendNote = useCallback(() => {
    if (canSend) {
      onTell(trimmed);
    }
  }, [canSend, onTell, trimmed]);
  const approve = useCallback(() => {
    if (approvable && !busy) {
      onApprove();
    }
  }, [approvable, busy, onApprove]);

  // The box takes the keyboard when it arrives in place of the composer,
  // whose box held it, but never from something else the person is in.
  useEffect(() => {
    const section = sectionRef.current;
    if (section === null || typeof document === "undefined") {
      return;
    }
    const active = document.activeElement;
    const thread = section.closest("[data-slot=assistant-thread]");
    if (
      active === null ||
      active === document.body ||
      (thread !== null && thread.contains(active))
    ) {
      section.focus({ preventScroll: true });
    }
  }, []);

  const closeNote = useCallback(() => {
    note.close();
    sectionRef.current?.focus({ preventScroll: true });
  }, [note]);

  const onKeyDown = (event: React.KeyboardEvent<HTMLElement>) => {
    // A dialog the box opened is a portal: its keys reach this handler
    // through React but are not the box's to answer.
    if (!(event.target instanceof Node) || !event.currentTarget.contains(event.target)) {
      return;
    }
    if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
      event.preventDefault();
      if (note.telling) {
        sendNote();
      } else {
        approve();
      }
      return;
    }
    if (event.key === "Escape" && canTell) {
      event.preventDefault();
      event.stopPropagation();
      if (note.telling) {
        closeNote();
      } else {
        note.open();
      }
      return;
    }
    if (event.altKey && !event.metaKey && !event.ctrlKey && event.code === "KeyL") {
      event.preventDefault();
      onDefer();
    }
  };

  return (
    <section
      ref={sectionRef}
      tabIndex={-1}
      aria-labelledby={titleId}
      data-slot="approval-dock"
      onKeyDown={onKeyDown}
      className="ui-lift bg-card ring-foreground/10 flex min-w-0 flex-col overflow-hidden rounded-lg ring-1 outline-none"
    >
      <Collapsible
        open={detailsOpen}
        onOpenChange={setDetailsOpen}
        className="flex min-w-0 flex-col"
      >
        <div className="flex min-w-0 flex-col gap-1 px-3 pt-2.5">
          <header className="flex min-w-0 items-center gap-2">
            <h3 className="min-w-0 flex-1 truncate text-sm">
              <span id={titleId} className="font-semibold">
                {title}
              </span>
              {subject !== "" && <span className="text-foreground-muted"> {subject}</span>}
            </h3>
            {figure !== null && (
              <span className="shrink-0 font-mono text-sm tabular-nums">{figure}</span>
            )}
            {severity && (
              <Badge variant={SEVERITY_BADGE[severity.tone]} className="shrink-0">
                {severity.label}
              </Badge>
            )}
            {permanent && (
              <Badge variant="warning" className="shrink-0">
                <TriangleAlertIcon aria-hidden />
                {t("Permanent")}
              </Badge>
            )}
            {count !== undefined && (
              <span className="text-foreground-muted shrink-0 text-xs tabular-nums">{count}</span>
            )}
            {total > 1 && (
              <span className="text-foreground-muted shrink-0 text-xs tabular-nums">
                {t("{0} of {1}", position, total)}
              </span>
            )}
          </header>
          {byline}
          <div className="text-foreground-muted line-clamp-2 text-xs leading-snug">{summary}</div>
          <CollapsibleTrigger
            render={
              <Button
                size="xxs"
                variant="ghost"
                className="text-foreground-muted -ml-2 self-start"
              />
            }
          >
            <ChevronRightIcon
              aria-hidden
              className={cn("size-3 transition-transform duration-200", detailsOpen && "rotate-90")}
            />
            {t("Details")}
          </CollapsibleTrigger>
        </div>
        <CollapsibleContent
          keepMounted={keepDetailsMounted}
          className={cn(
            "scrollbar-overlay min-w-0 overflow-y-auto px-3 pt-1",
            compact ? "max-h-[min(40vh,20rem)]" : "max-h-[min(45vh,28rem)]",
          )}
        >
          <div className="flex min-w-0 flex-col gap-2.5 pb-1">{details}</div>
        </CollapsibleContent>
      </Collapsible>

      <div className="flex min-w-0 flex-col gap-2 px-3 pt-2 empty:hidden">{attention}</div>

      {note.telling && (
        <div className="flex flex-col gap-1.5 px-3 pt-2">
          <label htmlFor={noteId} className="text-foreground-muted text-xs">
            {t("What should the agent do instead? It reads this and answers here.")}
          </label>
          <Textarea
            id={noteId}
            autoFocus
            value={note.note}
            maxLength={MAX_DECISION_NOTE_LENGTH}
            minRows={2}
            maxRows={compact ? 4 : 6}
            disabled={busy}
            placeholder={t("Tell the agent what to change, or why not…")}
            onChange={(event) => note.setNote(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter" && !event.shiftKey && !event.metaKey && !event.ctrlKey) {
                event.preventDefault();
                sendNote();
              }
            }}
          />
          {note.note.length > MAX_DECISION_NOTE_LENGTH * 0.9 && (
            <span className="text-foreground-subtle self-end text-xs tabular-nums">
              {t("{0} of {1} characters", note.note.length, MAX_DECISION_NOTE_LENGTH)}
            </span>
          )}
        </div>
      )}

      <footer className="mt-2 flex flex-wrap items-center gap-1.5 px-3 pb-2.5">
        {note.telling ? (
          <>
            <Button
              size="sm"
              onClick={sendNote}
              disabled={!canSend}
              isLoading={pending === "tell"}
              aria-keyshortcuts="Enter"
            >
              {t("Send to the agent")}
              <KeyHint compact={compact}>↵</KeyHint>
            </Button>
            <Button
              size="sm"
              variant="ghost"
              onClick={closeNote}
              disabled={busy}
              aria-keyshortcuts="Escape"
            >
              {t("Cancel")}
              <KeyHint compact={compact}>Esc</KeyHint>
            </Button>
          </>
        ) : (
          <>
            <Button
              size="sm"
              onClick={approve}
              disabled={busy || !approvable}
              isLoading={pending === "approve"}
              aria-keyshortcuts="Meta+Enter Control+Enter"
            >
              {approveLabel}
              <KeyHint compact={compact}>{formatShortcut("↵")}</KeyHint>
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={onReject}
              disabled={busy}
              isLoading={pending === "reject"}
            >
              {rejectLabel}
            </Button>
            {canTell && (
              <Button
                size="sm"
                variant="ghost"
                onClick={() => note.open()}
                disabled={busy}
                aria-keyshortcuts="Escape"
              >
                {t("Tell the agent")}
                <KeyHint compact={compact}>Esc</KeyHint>
              </Button>
            )}
          </>
        )}
        <span className="ml-auto flex items-center gap-0.5">
          {onModify && !note.telling && (
            <IconAction
              icon={PencilIcon}
              label={t("Modify")}
              onClick={onModify}
              disabled={busy || modifyDisabled}
            />
          )}
          <IconAction
            icon={ClockIcon}
            label={t("Decide later")}
            shortcut={formatAltShortcut("L")}
            keyshortcuts="Alt+L"
            onClick={onDefer}
            disabled={busy}
          />
        </span>
      </footer>
      {overlay}
    </section>
  );
}

/**
 * The key a button answers to, inside it and quieter than its words; left
 * out where the box is narrow. The button names the key for assistive tech
 * through `aria-keyshortcuts`, so the hint is not read twice.
 */
function KeyHint({ compact, children }: { compact: boolean; children: ReactNode }) {
  if (compact) {
    return null;
  }

  return (
    <kbd aria-hidden className="text-2xs -mr-0.5 font-sans opacity-60">
      {children}
    </kbd>
  );
}

/** A secondary answer as an icon, named by its tooltip and its label. */
function IconAction({
  icon: Icon,
  label,
  shortcut,
  keyshortcuts,
  onClick,
  disabled,
}: {
  icon: LucideIcon;
  label: string;
  shortcut?: string;
  keyshortcuts?: string;
  onClick: () => void;
  disabled: boolean;
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            size="icon-sm"
            variant="ghost"
            aria-label={label}
            aria-keyshortcuts={keyshortcuts}
            onClick={onClick}
            disabled={disabled}
            className="text-foreground-muted"
          />
        }
      >
        <Icon className="size-3.5" />
      </TooltipTrigger>
      <TooltipContent className="flex items-center gap-1.5">
        {label}
        {shortcut !== undefined && <Kbd>{shortcut}</Kbd>}
      </TooltipContent>
    </Tooltip>
  );
}

/** What every kind of decision does once the server has recorded it. */
function useAfterDecision(
  threadId: string,
  entry: ApprovalEntry,
  onDecided?: DockEntryProps["onDecided"],
) {
  const queryClient = useQueryClient();
  const followUp = useDecisionFollowUp();

  return useCallback(
    async (anchorId: string) => {
      await invalidateProposalViews(queryClient, threadId);
      // The server starts the turn in which the agent answers; the
      // conversation picks it up rather than sending anything itself, so a
      // note to the agent is answered once, by that turn.
      followUp?.(anchorId);
      onDecided?.(entry);
    },
    [entry, followUp, onDecided, queryClient, threadId],
  );
}

type DockAction = { action: "approve" | "reject" | "tell"; note?: string };

function ProposalDock({
  threadId,
  entry,
  position,
  total,
  compact,
  canTell,
  onDefer,
  onDecided,
}: DockEntryProps & { entry: Extract<ApprovalEntry, { kind: "proposal" }> }) {
  const t = useT();
  const queryClient = useQueryClient();
  const { proposal } = entry;
  const view = presentProposal(proposal);
  const fields = useMemo(() => proposal.fields ?? [], [proposal.fields]);
  const note = useDockNote();
  const afterDecision = useAfterDecision(threadId, entry, onDecided);
  const [editor, setEditor] = useState<ProposalEditorRequest | null>(null);

  // A write over a set of records lists them here, ticked, and approving
  // with some unticked approves the narrower set: its own preview is read,
  // and its digest is what the approval sends.
  const subsetField = fields.find(
    (field) => field.kind === "RecordSubset" && field.readOnly !== true,
  );
  const [subsetDraft, setSubsetDraft] = useState(() =>
    subsetField
      ? (draftFromArguments([subsetField], proposal.arguments)[subsetField.name] ?? "")
      : "",
  );
  const subsetChanges = useMemo(
    () =>
      subsetField
        ? changedValues([subsetField], proposal.arguments, { [subsetField.name]: subsetDraft })
        : {},
    [proposal.arguments, subsetDraft, subsetField],
  );
  const subsetValid = useMemo(
    () =>
      subsetField === undefined ||
      Object.keys(validateDraft([subsetField], { [subsetField.name]: subsetDraft })).length === 0,
    [subsetDraft, subsetField],
  );
  const narrowed = Object.keys(subsetChanges).length > 0;

  const proposedQuery = useProposalPreview({ scope: "mine", id: proposal.id });
  const proposedApproval = useApprovalGate(proposedQuery);
  const draft = useDraftPreview({
    target: narrowed ? { scope: "mine", proposalId: proposal.id } : undefined,
    modifications: narrowed && subsetValid ? subsetChanges : null,
  });
  const approval = narrowed ? draft.approval : proposedApproval;
  const gate = narrowed ? draft.gate : proposedApproval.gate;
  const shownQuery = narrowed ? draft.previewQuery : proposedQuery;
  const approvable = subsetValid && canApprove(gate) && !(narrowed && draft.refused);

  const outcomes = useMemo(
    () =>
      subsetField
        ? previewOutcomes([proposedQuery.data, narrowed ? draft.previewQuery.data : undefined], t)
        : NO_OUTCOMES,
    [draft.previewQuery.data, narrowed, proposedQuery.data, subsetField, t],
  );

  const decideMutation = useMutation({
    mutationFn: async ({ action, note: text }: DockAction): Promise<ProposalDecision> => {
      if (action === "approve") {
        const decision: ProposalDecision = narrowed ? "Modified" : "Accepted";
        await decideMyProposal(proposal.id, {
          decision,
          modifications: narrowed ? subsetChanges : undefined,
          reasonCode: "",
          previewDigest: gateDigest(gate),
        });
        return decision;
      }
      await decideMyProposal(proposal.id, {
        decision: "Rejected",
        reasonCode: "",
        previewDigest: gateDigest(proposedApproval.gate),
        note: text,
      });
      return "Rejected";
    },
    onSuccess: async (decision) => {
      markProposalDecided(queryClient, proposal.id, decision);
      await afterDecision(proposal.id);
    },
    onError: (error) => {
      // A digest that no longer matches is not a failure to report: the
      // write would now do something else, so the box reads it again and
      // says so, and the person decides on what is there now.
      if (!approval.handleDecisionError(error)) {
        handleMutationError({ error, resourceName: "Proposal" });
      }
      void invalidateProposalViews(queryClient, threadId);
    },
  });
  const act = (request: DockAction) => {
    approval.acknowledge();
    decideMutation.mutate(request);
  };

  const editable = fields.length > 0;
  const openEditor = (focus?: EditorFocus) =>
    setEditor({
      summary: view.summary,
      fields,
      arguments: proposal.arguments,
      focus,
      preview: { scope: "mine", proposalId: proposal.id },
      onConfirm: async (modifications, _reason, previewDigest) => {
        await decideMyProposal(proposal.id, {
          decision: "Modified",
          modifications,
          reasonCode: "",
          previewDigest,
        });
        markProposalDecided(queryClient, proposal.id, "Modified");
        await afterDecision(proposal.id);
      },
    });

  // A write that would be refused names what is wrong and offers the two ways
  // forward: change the value here, or tell the agent with the reasons
  // written, which turns it down and lets the agent propose it again.
  const wouldFail: WouldFailActions = {
    canChange: editable ? (param) => editableParam(param, fields) : undefined,
    onChange:
      editable && gate.state !== "stale"
        ? (reason) => openEditor({ param: reason.param, label: reason.label })
        : undefined,
    onAskAgent: canTell
      ? (reasons) => note.open(askAgentMessage(proposal.toolName, reasons, t))
      : undefined,
  };

  const shown = shownQuery.data;
  const refused = narrowed && draft.refused;
  const tally = subsetField
    ? subsetTally(subsetField, proposal.arguments?.[subsetField.name], subsetDraft)
    : null;

  return (
    <DockFrame
      title={view.title}
      subject={shown ? previewSubject(shown) : ""}
      figure={shown ? previewFigure(shown) : null}
      count={
        subsetField && tally
          ? subsetCountLabel(subsetField.resource, tally.kept, tally.total, t)
          : undefined
      }
      severity={view.severity}
      position={position}
      total={total}
      compact={compact}
      permanent={!view.reversible}
      byline={<ProposedBy agentId={proposal.agentId} agentName={proposal.agentName} />}
      summary={
        <DockSummary
          preview={shown}
          loading={shown === undefined && !shownQuery.isError}
          fallback={view.summary}
          worded={hasPresenter(proposal.toolName)}
        />
      }
      attention={
        refused ? (
          <Alert size="sm" variant="destructive">
            <AlertDescription>
              {t(
                "The records you kept would not go through as they are. Keep others, or reject it.",
              )}
            </AlertDescription>
          </Alert>
        ) : (
          <PreviewLoadState query={shownQuery} changed={approval.changed} loading={null}>
            {(preview) => <PreviewAttention preview={preview} wouldFail={wouldFail} />}
          </PreviewLoadState>
        )
      }
      details={
        <>
          {!refused &&
            (shown ? (
              <ProposalPreview
                preview={shown}
                density="compact"
                attention={false}
                wouldFail={wouldFail}
              />
            ) : shownQuery.isError ? (
              <Highlights highlights={view.highlights} />
            ) : (
              <PreviewSkeleton density="compact" />
            ))}
          {subsetField && (
            <SubsetSection
              field={subsetField}
              proposed={proposal.arguments?.[subsetField.name]}
              value={subsetDraft}
              outcomes={outcomes}
              onChange={setSubsetDraft}
            />
          )}
          {proposal.rationale !== "" && (
            <p className="text-foreground-muted text-xs leading-relaxed whitespace-pre-wrap">
              {proposal.rationale}
            </p>
          )}
        </>
      }
      overlay={<ProposalEditor request={editor} onClose={() => setEditor(null)} />}
      approveLabel={narrowed ? t("Approve the kept records") : t("Approve")}
      rejectLabel={t("Reject")}
      canApprove={approvable}
      pending={decideMutation.isPending ? (decideMutation.variables?.action ?? null) : null}
      onApprove={() => act({ action: "approve" })}
      onReject={() => act({ action: "reject" })}
      onTell={(text) => act({ action: "tell", note: text })}
      onModify={editable ? () => openEditor() : undefined}
      modifyDisabled={gate.state === "stale"}
      canTell={canTell}
      note={note}
      onDefer={onDefer}
    />
  );
}

/**
 * The decision in a sentence: the preview's own, which the tool writes from
 * the world as it is now, or the client's wording while it loads. A tool the
 * client has no words for waits for the preview rather than saying "Run …
 * with the values below" about values that are folded away.
 */
function DockSummary({
  preview,
  loading,
  fallback,
  worded,
}: {
  preview: ProposalPreviewData | undefined;
  loading: boolean;
  fallback: string;
  worded: boolean;
}) {
  const stated = preview?.summary.trim() ?? "";
  if (stated !== "") {
    return stated;
  }
  if (loading && !worded) {
    return <Skeleton className="my-0.5 h-3 w-2/3" />;
  }

  return fallback;
}

function SubsetSection({
  field,
  proposed,
  value,
  outcomes,
  onChange,
}: {
  field: NonNullable<AssistantProposal["fields"]>[number];
  proposed: unknown;
  value: string;
  outcomes: ReadonlyMap<string, string>;
  onChange: (value: string) => void;
}) {
  const t = useT();
  const labelId = useId();

  return (
    <div className="flex flex-col gap-1.5">
      <span id={labelId} className="text-foreground-subtle text-xs font-medium">
        {t("{0}: untick any to leave out", field.label)}
      </span>
      <RecordSubsetField
        labelId={labelId}
        field={field}
        proposed={proposed}
        value={value}
        outcomes={outcomes}
        readOnly={false}
        onChange={onChange}
      />
    </div>
  );
}

function PlanDock({
  threadId,
  entry,
  position,
  total,
  compact,
  canTell,
  onDefer,
  onDecided,
}: DockEntryProps & { entry: Extract<ApprovalEntry, { kind: "plan" }> }) {
  const t = useT();
  const queryClient = useQueryClient();
  const { plan, steps } = entry;
  const note = useDockNote();
  const afterDecision = useAfterDecision(threadId, entry, onDecided);

  // Every pending step previewed in order, each starting from what the steps
  // before it leave. One digest covers them all and goes with the approval.
  const previewQuery = usePlanPreview({ scope: "mine", id: plan.id });
  const approval = useApprovalGate(previewQuery);
  const previews = useMemo(() => previewsByStep(previewQuery.data), [previewQuery.data]);

  const decideMutation = useMutation({
    mutationFn: async ({ action, note: text }: DockAction) => {
      const decision = action === "approve" ? "Accepted" : "Rejected";
      await decideMyPlan(plan.id, {
        decision,
        reasonCode: "",
        previewDigest: gateDigest(approval.gate),
        note: action === "approve" ? undefined : text,
      });
      return decision;
    },
    onSuccess: async (decision) => {
      markPlanDecided(queryClient, plan.id, decision);
      await afterDecision(steps[0]?.id ?? plan.id);
    },
    onError: (error) => {
      if (!approval.handleDecisionError(error)) {
        handleMutationError({ error, resourceName: "Plan" });
      }
      void invalidateProposalViews(queryClient, threadId);
    },
  });
  const act = (request: DockAction) => {
    approval.acknowledge();
    decideMutation.mutate(request);
  };

  // A plan is decided whole, so a step that would be refused is not changed
  // here; the person tells the agent, with the reasons written.
  const wouldFail: WouldFailActions = {
    onAskAgent: canTell
      ? (reasons) => note.open(askAgentPlanMessage(plan.title, reasons, t))
      : undefined,
  };

  return (
    <DockFrame
      title={plan.title}
      count={t("{0, plural, one {# change} other {# changes}}", plan.stepCount)}
      position={position}
      total={total}
      compact={compact}
      permanent={steps.some((step) => !presentProposal(step).reversible)}
      byline={<ProposedBy agentId={plan.agentId} agentName={plan.agentName} />}
      summary={
        plan.summary !== ""
          ? plan.summary
          : t("{0, plural, one {# change, in order} other {# changes, in order}}", plan.stepCount)
      }
      attention={
        <PreviewLoadState query={previewQuery} changed={approval.changed} loading={null}>
          {(preview) => (
            <>
              {preview.stale && !preview.steps.some((step) => step.preview.stale) && (
                <StaleNotice missing={false} />
              )}
              {preview.steps
                .filter((step) => previewNeedsAttention(step.preview, { inPlan: true }))
                .map((step) => (
                  <div key={step.proposalId} className="flex min-w-0 flex-col gap-1.5">
                    <span className="text-foreground-muted text-xs">
                      {t("Step {0}", step.step)}
                    </span>
                    <PreviewAttention preview={step.preview} inPlan wouldFail={wouldFail} />
                  </div>
                ))}
            </>
          )}
        </PreviewLoadState>
      }
      details={
        steps.length > 0 && (
          <StepList
            steps={steps}
            settled={false}
            previews={previews}
            attention={false}
            wouldFail={wouldFail}
          />
        )
      }
      approveLabel={t("Approve all {0}", plan.stepCount)}
      rejectLabel={t("Reject all")}
      canApprove={canApprove(approval.gate)}
      pending={decideMutation.isPending ? (decideMutation.variables?.action ?? null) : null}
      onApprove={() => act({ action: "approve" })}
      onReject={() => act({ action: "reject" })}
      onTell={(text) => act({ action: "tell", note: text })}
      canTell={canTell}
      note={note}
      onDefer={onDefer}
    />
  );
}

type BatchOutcome = { approved: number; total: number; errors: string[] };

function BatchDock({
  threadId,
  entry,
  position,
  total,
  compact,
  canTell,
  onDefer,
  onDecided,
}: DockEntryProps & { entry: Extract<ApprovalEntry, { kind: "batch" }> }) {
  const t = useT();
  const queryClient = useQueryClient();
  const { proposals } = entry;
  const ids = useMemo(() => proposals.map((proposal) => proposal.id), [proposals]);
  const note = useDockNote();
  const afterDecision = useAfterDecision(threadId, entry, onDecided);

  // A few changes are read open; past that each opens on demand. Only the
  // previews actually open on screen send a digest: the rows live behind
  // "Details" and read nothing until it is opened, so one the person never
  // opened goes without one and is recorded as approved unreviewed, which
  // is the truth. A preview once on screen keeps its digest when Details
  // folds again.
  const [open, setOpen] = useState<ReadonlySet<string>>(
    () => new Set(proposals.length <= BATCH_OPEN_LIMIT ? ids : []),
  );
  const [shown, setShown] = useState<ReadonlyMap<string, string>>(() => new Map());
  const [loading, setLoading] = useState<ReadonlySet<string>>(() => new Set());
  const report = useCallback((id: string, digest: string | null, isLoading: boolean) => {
    setShown((current) => {
      if ((current.get(id) ?? null) === digest) {
        return current;
      }
      const next = new Map(current);
      if (digest === null) {
        next.delete(id);
      } else {
        next.set(id, digest);
      }
      return next;
    });
    setLoading((current) => {
      if (current.has(id) === isLoading) {
        return current;
      }
      const next = new Set(current);
      if (isLoading) {
        next.add(id);
      } else {
        next.delete(id);
      }
      return next;
    });
  }, []);
  const leave = useCallback((id: string) => {
    setLoading((current) => {
      if (!current.has(id)) {
        return current;
      }
      const next = new Set(current);
      next.delete(id);
      return next;
    });
  }, []);
  const toggle = (id: string) =>
    setOpen((current) => {
      const next = new Set(current);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });

  const unreviewed = ids.filter((id) => !shown.has(id)).length;

  const decideMutation = useMutation({
    mutationFn: async ({ action, note: text }: DockAction): Promise<BatchOutcome> => {
      const approving = action === "approve";
      const results = await decideMyProposals(ids, {
        decision: approving ? "Accepted" : "Rejected",
        reasonCode: approving ? undefined : BATCH_REJECT_REASON,
        previewDigests: approving ? batchPreviewDigests(ids, shown) : undefined,
        note: approving ? undefined : text,
      });
      const failed = results.filter((result) => (result.error ?? "") !== "");
      for (const result of results) {
        if ((result.error ?? "") === "" || result.decision) {
          markProposalDecided(queryClient, result.proposalId, approving ? "Accepted" : "Rejected");
        }
      }

      return {
        approved: results.length - failed.length,
        total: results.length,
        errors: failed.map((result) => {
          const proposal = proposals.find((candidate) => candidate.id === result.proposalId);
          const summary = proposal ? presentProposal(proposal).summary : "";
          return summary === "" ? (result.error ?? "") : `${summary}: ${result.error ?? ""}`;
        }),
      };
    },
    onSuccess: async (outcome) => {
      if (outcome.errors.length > 0) {
        toast.warning(t("{0} of {1} went through", outcome.approved, outcome.total), {
          description: outcome.errors.join(" · "),
        });
      }
      await afterDecision(ids[0]);
    },
    onError: (error) => {
      handleMutationError({ error, resourceName: "Proposals" });
      void invalidateProposalViews(queryClient, threadId);
    },
  });

  const first = proposals[0];
  const title = first ? presentProposal(first).title : "";
  const stillLoading = ids.some((id) => open.has(id) && loading.has(id));

  const worded = first !== undefined && hasPresenter(first.toolName);

  return (
    <DockFrame
      title={title}
      count={t("{0, plural, one {# change} other {# changes}}", proposals.length)}
      position={position}
      total={total}
      compact={compact}
      permanent={proposals.some((proposal) => !presentProposal(proposal).reversible)}
      byline={first ? <ProposedBy agentId={first.agentId} agentName={first.agentName} /> : null}
      summary={
        worded
          ? proposals.map((proposal) => presentProposal(proposal).summary).join(" · ")
          : t(
              "{0, plural, one {# change of one kind, decided together} other {# changes of one kind, decided together}}",
              proposals.length,
            )
      }
      attention={
        unreviewed > 0 && (
          <p className="text-foreground-muted text-xs">
            {t(
              "{0, plural, one {# of them is not open; approving records it as approved without reviewing what it changes.} other {# of them are not open; approving records them as approved without reviewing what they change.}}",
              unreviewed,
            )}
          </p>
        )
      }
      details={
        <ul className="flex flex-col gap-1.5">
          {proposals.map((proposal) => (
            <BatchRow
              key={proposal.id}
              proposal={proposal}
              open={open.has(proposal.id)}
              onToggle={() => toggle(proposal.id)}
              onReport={report}
              onLeave={leave}
            />
          ))}
        </ul>
      }
      approveLabel={t("Approve all {0}", proposals.length)}
      rejectLabel={t("Reject all")}
      canApprove={!stillLoading}
      pending={decideMutation.isPending ? (decideMutation.variables?.action ?? null) : null}
      onApprove={() => decideMutation.mutate({ action: "approve" })}
      onReject={() => decideMutation.mutate({ action: "reject" })}
      onTell={(text) => decideMutation.mutate({ action: "tell", note: text })}
      canTell={canTell}
      note={note}
      onDefer={onDefer}
    />
  );
}

/**
 * One change of several decided together: its sentence, opening onto what it
 * would do. It reports the digest of the preview it has on screen, which is
 * the only digest the approval sends for it, and stops counting as loading
 * once it leaves the screen.
 */
function BatchRow({
  proposal,
  open,
  onToggle,
  onReport,
  onLeave,
}: {
  proposal: AssistantProposal;
  open: boolean;
  onToggle: () => void;
  onReport: (id: string, digest: string | null, loading: boolean) => void;
  onLeave: (id: string) => void;
}) {
  const view = presentProposal(proposal);
  const query = useProposalPreview({ scope: "mine", id: proposal.id, enabled: open });
  const digest = open && query.data !== undefined ? query.data.digest : null;
  const loading = open && query.data === undefined && !query.isError;

  useEffect(() => {
    onReport(proposal.id, digest, loading);
  }, [digest, loading, onReport, proposal.id]);
  useEffect(() => () => onLeave(proposal.id), [onLeave, proposal.id]);

  return (
    <li className="flex min-w-0 flex-col gap-1.5">
      <button
        type="button"
        aria-expanded={open}
        onClick={onToggle}
        className="hover:bg-surface-hover ui-focus-ring -mx-1.5 flex min-w-0 items-center gap-1.5 rounded-md px-1.5 py-1 text-left text-sm transition-colors"
      >
        <ChevronRightIcon
          aria-hidden
          className={cn(
            "text-foreground-subtle size-3.5 shrink-0 transition-transform duration-200",
            open && "rotate-90",
          )}
        />
        <span className="min-w-0 flex-1 truncate">{view.summary}</span>
      </button>
      {open && (
        <div className="pl-5">
          <PreviewLoadState
            query={query}
            changed={false}
            density="compact"
            fallback={<Highlights highlights={view.highlights} />}
          >
            {(preview) => <ProposalPreview preview={preview} density="compact" />}
          </PreviewLoadState>
        </div>
      )}
    </li>
  );
}
