import { AssistantAgentProvider } from "@/components/agent-identity/agent-context";
import { stepsFromExchanges, type ToolStep } from "@/components/assistant/activity";
import type { ApprovalEntry } from "@/components/assistant/approval-queue";
import { ArtifactOpenerProvider } from "@/components/assistant/artifact-opener";
import { ArtifactLinkContext, type ArtifactLinkRenderer } from "@/components/elements/ai-markdown";
import { keptWhileWriting, withArtifactRefs } from "@/lib/artifact-ref";
import { DecisionFollowUpProvider } from "@/components/assistant/decision-follow-up";
import { fillCommand, SLASH_COMMANDS } from "@/components/assistant/composer-commands";
import { readyAttachments } from "@/components/assistant/composer";
import { useAskableAgent } from "@/components/assistant/use-askable-agent";
import { useComposerContext } from "@/components/assistant/use-composer-context";
import { useOpeningQuestion } from "@/components/assistant/use-opening-question";
import { useThreadModel } from "@/components/assistant/use-thread-model";
import { meterView } from "@/components/assistant/compaction";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { useAssistantStore } from "@/stores/assistant-store";
import { useDeskHandoffStore } from "@/stores/desk-handoff-store";
import { useDeskStore } from "@/stores/desk-store";
import type { AssistantArtifact, AssistantProposal, AssistantThread } from "@/types/assistant";
import { invalidateProposalViews } from "@/lib/proposal-cache";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort, formatUnixTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useReducedMotion } from "motion/react";
import { useNavigate } from "react-router";
import {
  Fragment,
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type KeyboardEvent,
} from "react";
import { DeskWorkspace } from "./artifacts/desk-workspace";
import { withoutPendingLookups } from "./artifacts/pending-lookups";
import { useDeskAttachments } from "./composer/desk-attachments";
import { usePoorlyReadFiles } from "./conversation/desk-poorly-read";
import { useOnline } from "./desk-online";
import { useDeskScans } from "./composer/desk-capture";
import { DeskComposer } from "./composer/desk-composer";
import { DeskModelPicker } from "./composer/desk-model-picker";
import { DeskContextMeter } from "./composer/desk-context-meter";
import { DeskCompactMark } from "./conversation/desk-compact-mark";
import { DeskPageChip, useDeskPage } from "./composer/desk-page-chip";
import { DeskDropOverlay, useDeskDrop } from "./composer/desk-uploads";
import { DeskApprovalCard, DeskApprovedCard } from "./conversation/desk-approval-card";
import { DeskUndoBar, useUndoWindow } from "./conversation/desk-undo-bar";
import { DeskFactsBar } from "./composer/desk-facts";
import { DeskHandoffCard } from "./handoff/desk-handoff-card";
import { useDeskSetting } from "@/stores/desk-settings-store";
import { DeskConfetti } from "./conversation/desk-confetti";
import {
  DeskCutOffCard,
  DeskFallbackLine,
  DeskNoModelCard,
  DeskPoorlyReadCard,
} from "./conversation/desk-failures";
import { DeskWriteResultCard } from "./conversation/desk-tool-failures";
import { DeskScheduleCard } from "./conversation/desk-schedule-card";
import {
  isScheduleRequest,
  useCreateSchedule,
  useScheduleActions,
} from "./conversation/desk-schedules";
import {
  DeskInlineArtifact,
  DeskQuestion,
  withoutCutNote,
  DeskReply,
  DeskRow,
  DeskStreamingReply,
} from "./conversation/desk-turns";
import { composerStatus, streamingText } from "./conversation/turn-status";
import { useStickToBottom } from "./conversation/use-stick-to-bottom";
import { DeskErrorButton, DeskErrorCard } from "./desk-error-card";
import { DeskIcon } from "./desk-icons";
import { deskComposerLock, DeskUsageMeter, useRequestMore } from "./desk-locks";
import { useDesk } from "./desk-layout";
import { DeskTermsNote } from "./desk-terms-note";

export type DeskConversationProps = {
  thread: AssistantThread;
  agent: AgentChoice | null;
  agentsUnavailable: boolean;
  onStartNew?: () => void;
};

const NO_STEPS: ToolStep[] = [];
const NO_ARTIFACTS: AssistantArtifact[] = [];

/** How long a confetti burst lasts. */
const CONFETTI_MS = 5400;

/** The reason a saved closing note gives, without its italics and its stock opening. */
function closingReason(content: string): string {
  const plain = content.replace(/^_|_$/g, "").trim();
  return plain.replace(/^This reply failed before it started\. Ask again to continue\.\s*/u, "");
}

function turnTime(at: number, timezone: string, t: TranslateFn): string {
  const today = new Date().toLocaleDateString("en-CA", { timeZone: timezone });
  const day = new Date(at * 1000).toLocaleDateString("en-CA", { timeZone: timezone });
  if (day === today) {
    return formatUnixTime(at, { timezone }) || t("Now");
  }

  return formatUnixDateTimeShort(at, { timezone });
}

/** A conversation nobody can send to takes no files either. */
function lockedForFiles(thread: AssistantThread, agentsUnavailable: boolean): boolean {
  return !thread.canContinue || agentsUnavailable;
}

/** Whether a saved reply has anything of its own to show besides the work behind it. */
function replyShows(
  entry: { message: { content: string }; tools: { call: { name: string } }[] },
  artifacts: number,
): boolean {
  return (
    entry.message.content !== "" ||
    artifacts > 0 ||
    entry.tools.some((tool) => tool.call.name === "ask_user" || tool.call.name === "run_report")
  );
}

/**
 * One conversation, as the Desk draws it: the person's questions as headings,
 * each reply under it with its time in the margin, and the composer with any
 * change waiting on them just above it. While the agent works, the room lights
 * and the composer says what it is doing; the reply appears only once its
 * first words do. The workspace slides in beside it and the conversation
 * narrows to make room.
 */
export function DeskConversation({
  thread,
  agent,
  agentsUnavailable,
  onStartNew,
}: DeskConversationProps) {
  const t = useT();
  const desk = useDesk();
  const reduceMotion = useReducedMotion();
  const queryClient = useQueryClient();
  const celebrate = useDeskSetting("celebrate");
  const motion = useDeskSetting("motion");
  const timezone = useAuthStore((state) => state.user?.timezone) || "UTC";
  const opening = useOpeningQuestion(thread.id);
  const artifactsQuery = useQuery(queries.assistant.artifacts(thread.id));
  const budgetQuery = useQuery({ ...queries.assistant.threadBudget(thread.id), staleTime: 60_000 });
  const artifacts = useMemo(() => artifactsQuery.data?.results ?? [], [artifactsQuery.data]);
  const activeArtifactId = useDeskStore((state) => state.activeArtifactByThread[thread.id] ?? null);
  const chapters = useDeskStore((state) => state.chaptersByThread[thread.id]);
  const toggleChapter = useDeskStore((state) => state.toggleChapter);

  const setActiveThreadId = useAssistantStore((state) => state.setActiveThreadId);
  const openWidget = useAssistantStore((state) => state.openWidget);
  const carryConversation = useCallback(() => {
    setActiveThreadId(thread.id);
    openWidget();
  }, [openWidget, setActiveThreadId, thread.id]);

  const deskPage = useDeskPage();
  const askable = useAskableAgent({ threads: desk.threads });
  const composerContext = useComposerContext(thread.id);
  const scans = useDeskScans(thread.id);
  const attachments = useDeskAttachments(composerContext, scans);

  // Files and records from the front page, handed over with the question
  // that started this conversation: the files upload here, and the question
  // waits for them before it goes.
  const handoff = useDeskHandoffStore((state) =>
    state.handoff?.threadId === thread.id ? state.handoff : null,
  );
  const setHandoff = useDeskHandoffStore((state) => state.setHandoff);
  const handedOver = useRef(false);
  const { attachFiles } = composerContext;
  useEffect(() => {
    if (!handoff || handedOver.current) {
      return;
    }
    handedOver.current = true;
    if (handoff.files.length > 0) {
      attachFiles(handoff.files);
    }
  }, [attachFiles, handoff]);
  const openingHold =
    handoff !== null &&
    (composerContext.attachments.length < handoff.files.length ||
      composerContext.attachments.some((item) => item.status === "uploading"));
  const openingPayload = useMemo(
    () =>
      handoff
        ? { attachments: readyAttachments(composerContext.attachments), mentions: handoff.mentions }
        : undefined,
    [composerContext.attachments, handoff],
  );
  // The question that started the conversation, shown the moment it was
  // asked: its files upload first and the question goes once they are in,
  // and until then the conversation would otherwise be an empty page.
  const waitingOpening = openingHold && opening.openingQuestion ? opening.openingQuestion : null;
  const waitingFiles = useMemo(
    () =>
      composerContext.attachments.map((item) => ({
        documentId: item.documentId ?? item.id,
        fileName: item.name,
        contentType: item.contentType,
        fileSize: item.size,
      })),
    [composerContext.attachments],
  );
  const { clear: clearComposerContext } = composerContext;
  const { clear: clearAttachments } = attachments;
  const { onOpeningQuestionSent } = opening;
  const openingSent = useCallback(() => {
    onOpeningQuestionSent();
    if (handoff) {
      setHandoff(null);
      clearComposerContext();
      clearAttachments();
    }
  }, [clearAttachments, clearComposerContext, handoff, onOpeningQuestionSent, setHandoff]);

  // A conversation started from the front page with "every weekday at 7:30,
  // …" is a schedule, not a question: it is kept here rather than handed to
  // the turn.
  const scheduling = useCreateSchedule(thread.id);
  const openingSchedule =
    opening.openingQuestion && isScheduleRequest(opening.openingQuestion)
      ? opening.openingQuestion
      : null;
  const scheduledOpening = useRef(false);
  const { create: createSchedule } = scheduling;
  useEffect(() => {
    if (openingSchedule === null || openingHold || scheduledOpening.current) {
      return;
    }
    scheduledOpening.current = true;
    createSchedule(openingSchedule);
    openingSent();
  }, [createSchedule, openingHold, openingSchedule, openingSent]);
  const schedulesQuery = useQuery(queries.assistant.schedules(thread.id));
  const scheduleById = useMemo(
    () => new Map((schedulesQuery.data?.items ?? []).map((item) => [item.id, item])),
    [schedulesQuery.data],
  );
  const scheduleActions = useScheduleActions(thread.id);
  const [clock, setClock] = useState(() => Math.floor(Date.now() / 1000));
  useEffect(() => {
    // "Today" and "Tomorrow" on a schedule's next run move at midnight; a
    // minute is close enough for a label.
    const timer = window.setInterval(() => setClock(Math.floor(Date.now() / 1000)), 60_000);
    return () => window.clearInterval(timer);
  }, []);

  const model = useThreadModel({
    thread,
    agent,
    agentsUnavailable,
    artifacts,
    onLiveArtifacts: desk.noteLiveArtifacts,
    onWorkingChange: desk.setWorking,
    onNavigate: carryConversation,
    followNavigation: false,
    openingQuestion: openingSchedule === null ? opening.openingQuestion : undefined,
    onOpeningQuestionSent: openingSent,
    openingHold,
    openingPayload,
    pageContextSource: deskPage.context,
  });
  const { entries, placements, turn, isActive, compaction } = model;

  // Each reply spends from the caps, so the warning above the composer is
  // read again once one ends rather than waiting out its staleness.
  const wasActive = useRef(isActive);
  const refetchBudget = budgetQuery.refetch;
  useEffect(() => {
    if (wasActive.current && !isActive) {
      void refetchBudget();
    }
    wasActive.current = isActive;
  }, [isActive, refetchBudget]);

  const { setProviderId } = model;
  const handoffProvider = handoff?.providerId ?? "";
  useEffect(() => {
    if (handoffProvider !== "") {
      setProviderId(handoffProvider);
    }
  }, [handoffProvider, setProviderId]);

  const { setArtifactCount, openArtifact: openInDesk, setWorkspaceOpen, workspaceOpen } = desk;
  // The running turn's lookups are not counted until the reply keeps them.
  const artifactTotal = useMemo(() => {
    const visible = withoutPendingLookups(
      artifacts,
      artifactsQuery.data?.counts,
      desk.pendingLookups,
    );
    return visible.counts?.all ?? visible.results.length;
  }, [artifacts, artifactsQuery.data, desk.pendingLookups]);
  useEffect(() => setArtifactCount(artifactTotal), [artifactTotal, setArtifactCount]);

  const togglePin = useCallback(
    (messageId: string) => toggleChapter(thread.id, messageId),
    [thread.id, toggleChapter],
  );
  const openArtifact = useCallback(
    (artifactId: string) => openInDesk(thread.id, artifactId),
    [openInDesk, thread.id],
  );

  const replies = entries.filter((entry) => entry.kind === "assistant").length;
  const ownMessages = entries.filter((entry) => entry.kind === "user").length + (turn ? 1 : 0);
  const { scrollRef, away, unread, jumpToLatest } = useStickToBottom({ replies, ownMessages });

  // A long conversation opens on its newest page; the earlier ones are read
  // as the person scrolls up to them, and the page holds still while they
  // arrive above what is being read.
  const olderRef = useRef<HTMLDivElement>(null);
  const keptHeight = useRef<number | null>(null);
  const { has: hasOlder, loading: loadingOlder, load: loadOlder } = model.older;
  useEffect(() => {
    const marker = olderRef.current;
    const scroller = scrollRef.current;
    if (!marker || !scroller || !hasOlder) {
      return;
    }
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry?.isIntersecting && !loadingOlder) {
          keptHeight.current = scroller.scrollHeight;
          loadOlder();
        }
      },
      { root: scroller, rootMargin: "400px 0px 0px 0px" },
    );
    observer.observe(marker);
    return () => observer.disconnect();
  }, [hasOlder, loadOlder, loadingOlder, scrollRef]);
  const messageCount = model.entries.length;
  useLayoutEffect(() => {
    const scroller = scrollRef.current;
    if (keptHeight.current === null || !scroller || loadingOlder) {
      return;
    }
    scroller.scrollTop += scroller.scrollHeight - keptHeight.current;
    keptHeight.current = null;
  }, [loadingOlder, messageCount, scrollRef]);

  const drag = useDeskDrop(lockedForFiles(thread, agentsUnavailable) ? null : attachments.add);
  useEffect(() => {
    const room = scrollRef.current?.closest(".dsk");
    room?.classList.toggle("dk-dragging", drag.on);
    return () => room?.classList.remove("dk-dragging");
  }, [drag.on, scrollRef]);
  const { onDraftChange } = model;
  const explainPage = useCallback(() => {
    const explain = SLASH_COMMANDS.find((command) => command.name === "explain");
    if (explain) {
      onDraftChange(fillCommand(explain, []));
    }
  }, [onDraftChange]);

  const [burst, setBurst] = useState<number | null>(null);
  const approveRef = useRef<(() => void) | null>(null);
  useEffect(() => {
    if (burst === null) {
      return;
    }
    const timer = window.setTimeout(() => setBurst(null), CONFETTI_MS);
    return () => window.clearTimeout(timer);
  }, [burst]);

  // An approval waits out its undo window on the server. When it goes
  // through, the conversation reads the outcome and picks up the turn in
  // which the agent reports it.
  const { followUpDecision } = model;
  const undo = useUndoWindow(thread.id, () => {
    void invalidateProposalViews(queryClient, thread.id);
    followUpDecision();
    if (!reduceMotion && celebrate === "confetti" && motion === "full") {
      setBurst(Date.now());
    }
  });
  const holding = undo.state.phase !== "idle";

  const reviewEntry = (entry: ApprovalEntry) => {
    const artifact = artifacts.find(
      (candidate) =>
        candidate.kind === "decision_request" &&
        ((candidate.proposalId !== "" && entry.members.includes(candidate.proposalId)) ||
          (entry.kind === "plan" && candidate.planId === entry.plan.id)),
    );
    if (artifact) {
      openArtifact(artifact.id);
    } else {
      setWorkspaceOpen(true);
    }
  };

  const onKeyDown = (event: KeyboardEvent) => {
    if ((event.metaKey || event.ctrlKey) && event.key === "Enter" && approveRef.current) {
      event.preventDefault();
      approveRef.current();
    }
    if (event.key === "Escape" && workspaceOpen) {
      setWorkspaceOpen(false);
    }
  };

  // For a reply that did not finish: what had arrived of it, and the question
  // it answered, so its card can count the words and ask again.
  const { arrivedBefore, questionBefore } = useMemo(() => {
    const arrived = new Map<string, string>();
    const asked = new Map<string, string>();
    let text = "";
    let question = "";
    for (const entry of entries) {
      if (entry.kind === "user") {
        text = "";
        question = entry.message.content;
        continue;
      }
      if (entry.kind !== "assistant") {
        continue;
      }
      asked.set(entry.message.id, question);
      if (entry.message.failure) {
        arrived.set(entry.message.id, text.trim());
        continue;
      }
      text += ` ${entry.message.content}`;
    }
    return { arrivedBefore: arrived, questionBefore: asked };
  }, [entries]);
  const ask = useCallback(
    (text: string) => void model.send(text, undefined, model.providerId),
    [model],
  );
  const retryFor = (id: string) => {
    const question = questionBefore.get(id) ?? "";
    return question !== "" ? () => ask(question) : undefined;
  };
  const navigate = useNavigate();
  const requestMore = useRequestMore(thread.id);
  const checkProviders = useCallback(
    () => void navigate("/admin/agent-control?tab=providers"),
    [navigate],
  );
  const vendorOf = useCallback(
    (providerId: string | null | undefined) =>
      providerId ? model.providers.find((option) => option.id === providerId)?.vendor : undefined,
    [model.providers],
  );
  // When no model answered, the question goes back in the composer, so the
  // card's "Your message is saved" holds and asking again is one press.
  const restoredFor = useRef<number | null>(null);
  useEffect(() => {
    if (
      turn?.status === "error" &&
      turn.failedProviders.length > 0 &&
      restoredFor.current !== turn.startedAt
    ) {
      restoredFor.current = turn.startedAt;
      if (model.draft.trim() === "") {
        onDraftChange(turn.userContent);
      }
    }
  }, [turn, model.draft, onDraftChange]);
  // Sent too fast: the question goes back in the composer and the send button
  // counts down; when it reaches the end the turn is cleared so it can go.
  const limitedFor = turn?.status === "error" ? turn.rateLimited : null;
  const dismissTurn = model.dismiss;
  useEffect(() => {
    if (limitedFor === null || !turn) {
      return;
    }
    if (model.draft.trim() === "") {
      onDraftChange(turn.userContent);
    }
    const timer = window.setTimeout(() => void dismissTurn(), limitedFor * 1000);
    return () => window.clearTimeout(timer);
    // The draft is read once, when the wait starts.
    // oxlint-disable-next-line react-hooks/exhaustive-deps
  }, [limitedFor, turn?.startedAt, dismissTurn, onDraftChange]);
  // Changes that ran but did not all go through, under the reply that drafted
  // them; and each turn's last reply, which carries the steps that failed.
  const writeResults = useMemo(() => {
    const out = new Map<string, AssistantProposal[]>();
    for (const proposal of model.proposalsQuery.data?.results ?? []) {
      const failedItems = proposal.executionResult?.failed?.length ?? 0;
      if (proposal.status !== "ExecutionFailed" && failedItems === 0) continue;
      const source = proposal.sourceMessageId;
      if (!source) continue;
      out.set(source, [...(out.get(source) ?? []), proposal]);
    }
    return out;
  }, [model.proposalsQuery.data]);
  const lastOfTurn = useMemo(() => {
    const out = new Set<string>();
    let last = "";
    for (const entry of entries) {
      if (entry.kind === "user") {
        if (last !== "") out.add(last);
        last = "";
      } else if (entry.kind === "assistant" && !entry.message.failure) {
        last = entry.message.id;
      }
    }
    if (last !== "" && turn === null) out.add(last);
    return out;
  }, [entries, turn]);
  // Files on a question that reading made out little of, keyed by the last
  // reply of that turn, under which the card asking for a clearer copy sits.
  const askedFiles = useMemo(
    () =>
      entries.flatMap((entry) => (entry.kind === "user" ? (entry.message.attachments ?? []) : [])),
    [entries],
  );
  const unreadable = usePoorlyReadFiles(askedFiles);
  const poorlyRead = useMemo(() => {
    const out = new Map<string, string[]>();
    let files: string[] = [];
    for (const entry of entries) {
      if (entry.kind === "user") {
        files = (entry.message.attachments ?? [])
          .filter((file) => unreadable.has(file.documentId))
          .map((file) => file.fileName);
      } else if (entry.kind === "assistant" && lastOfTurn.has(entry.message.id)) {
        if (files.length > 0) out.set(entry.message.id, files);
        files = [];
      }
    }
    return out;
  }, [entries, lastOfTurn, unreadable]);
  const [filePickerSignal, setFilePickerSignal] = useState(0);
  // The jump button floats just above the dock, whose height changes with
  // what sits over the composer: the terms note, an approval, a lock.
  const [dockHeight, setDockHeight] = useState<number | null>(null);
  const dockRef = useCallback((node: HTMLDivElement | null) => {
    if (!node) return;
    const observer = new ResizeObserver(() => setDockHeight(node.offsetHeight));
    observer.observe(node);
    return () => observer.disconnect();
  }, []);
  // A question asked while offline waits here, shown as sent, and goes the
  // moment the connection is back.
  const online = useOnline();
  const [queued, setQueued] = useState<{
    content: string;
    payload: Parameters<typeof model.send>[3];
  } | null>(null);
  const { send, providerId } = model;
  useEffect(() => {
    if (online && queued) {
      setQueued(null);
      void send(queued.content, undefined, providerId, queued.payload);
    }
  }, [online, queued, send, providerId]);
  // A question an artifact asked on the person's behalf goes as theirs, once
  // the conversation is free to take it.
  const pendingAsk = useDeskStore((state) => state.asks[thread.id] ?? null);
  const takeAsk = useDeskStore((state) => state.takeAsk);
  const busyForAsk = isActive;
  useEffect(() => {
    if (pendingAsk === null || busyForAsk || !online) {
      return;
    }
    const question = takeAsk(thread.id);
    if (question) {
      void send(question, undefined, providerId);
    }
  }, [busyForAsk, online, pendingAsk, providerId, send, takeAsk, thread.id]);
  const [pickerSignal, setPickerSignal] = useState(0);
  const [agentPickerSignal, setAgentPickerSignal] = useState(0);
  // A question the guard turned away: its card offers to ask another agent
  // or to rephrase, both starting from the question as it was asked.
  const refusedQuestion = useMemo(() => {
    const out = new Map<string, string>();
    let declined = "";
    for (const entry of entries) {
      if (entry.kind === "declined") {
        declined = entry.message.content;
      } else if (entry.kind === "refusal") {
        if (declined !== "") {
          out.set(entry.message.id, declined);
        }
        declined = "";
      }
    }
    return out;
  }, [entries]);
  const currentModelName =
    model.providers.find((option) => option.id === model.providerId)?.model ||
    model.providers[0]?.model ||
    t("this model");
  const lastIsRefusal = entries.at(-1)?.kind === "refusal" && turn === null;
  // Switching model mid-retry asks again: the retrying turn is stopped, the
  // question goes back in the composer, and the picker opens on it.
  const switchModel = useCallback(() => {
    const question = turn?.userContent ?? "";
    model.stop();
    void model.dismiss();
    if (question !== "") {
      model.onDraftChange(question);
    }
    setPickerSignal((value) => value + 1);
  }, [model, turn?.userContent]);
  const status = composerStatus(turn, t, switchModel);
  const live = streamingText(turn);
  const chapterOf = (id: string) => (chapters ? chapters.indexOf(id) + 1 : 0);
  const showCard = model.showDock && model.current !== null && !holding;
  // The change waiting on the person, or the approval in its undo window, is
  // attached to the top of the composer.
  const attached = showCard || holding;
  const pending = model.queue.length > 0;
  const jumping = away || unread > 0;

  const composerLock = deskComposerLock({
    thread,
    // A turned-off agent is no longer among the ones the person can pick, so
    // its name comes from the conversation's own record.
    agentName: agent?.name || budgetQuery.data?.agentName || t("This agent"),
    budget: budgetQuery.data,
    noModel: model.providersReady && model.providers.length === 0,
    timezone,
    t,
    requestMore: (kind) => void requestMore(kind),
    startElsewhere: () => void navigate("/desk"),
    openAgentControl: () => void navigate("/admin/agent-control?tab=providers"),
  });
  const fallbackLock =
    model.block === "read-only" ? (
      <>
        <DeskIcon name="lock" size={13} stroke={2} />
        <span>
          <b>{t("This conversation can no longer continue.")}</b> {t("You can still read it.")}
        </span>
      </>
    ) : model.block === "agents-unavailable" ? (
      <>
        <DeskIcon name="alert" size={13} stroke={2} />
        <span>
          <b>{t("The agents could not be loaded.")}</b> {t("Refresh to try again.")}
        </span>
      </>
    ) : model.block !== null ? (
      <>
        <DeskIcon name="lock" size={13} stroke={2} />
        <span>
          <b>{t("This agent has been turned off.")}</b> {t("You can still read this conversation.")}
        </span>
      </>
    ) : null;
  const lock = composerLock?.lock ?? fallbackLock;
  const rateLimited = turn?.status === "error" ? turn.rateLimited : null;

  const replySteps = useMemo(() => {
    const byEntry = new Map<string, ToolStep[]>();
    let steps: ToolStep[] = [];
    let askedAt = 0;
    for (const entry of entries) {
      if (entry.kind === "user") {
        steps = [];
        askedAt = entry.message.createdAt;
      } else if (entry.kind === "assistant") {
        steps = [...steps, ...stepsFromExchanges(entry.tools, askedAt)];
        byEntry.set(entry.message.id, steps);
      }
    }
    return byEntry;
  }, [entries]);

  // A step that only looked something up has no words of its own; what it
  // made is shown under the reply's text that follows it, where the design
  // puts a reply's artifacts, or on its last step when no text follows.
  const rowArtifacts = useMemo(() => {
    const byEntry = new Map<string, AssistantArtifact[]>();
    let carried: AssistantArtifact[] = [];
    let lastReply: string | null = null;
    const settle = () => {
      if (carried.length > 0 && lastReply !== null) {
        byEntry.set(lastReply, [...(byEntry.get(lastReply) ?? []), ...carried]);
      }
      carried = [];
      lastReply = null;
    };
    for (const entry of entries) {
      if (entry.kind === "user") {
        settle();
        continue;
      }
      if (entry.kind !== "assistant") {
        continue;
      }
      const own = model.artifactsByMessage.get(entry.message.id) ?? [];
      if (entry.message.content === "") {
        carried = [...carried, ...own];
      } else {
        byEntry.set(entry.message.id, [...carried, ...own]);
        carried = [];
      }
      lastReply = entry.message.id;
    }
    settle();
    return byEntry;
  }, [entries, model.artifactsByMessage]);

  // Which rows open the conversation, and which reply steps carry their
  // reply's time: worked out before drawing, from what each row will show.
  const layout = useMemo(() => {
    const first = new Set<string>();
    const headed = new Set<string>();
    let seenFirst = false;
    let replyHeaded = false;
    for (const entry of entries) {
      if (entry.kind === "decision") {
        continue;
      }
      if (entry.kind === "assistant") {
        const shown = replyShows(
          entry,
          (rowArtifacts.get(entry.message.id) ?? NO_ARTIFACTS).length,
        );
        if (!shown) {
          continue;
        }
        if (!replyHeaded || !(placements.get(entry.message.id)?.continued ?? false)) {
          headed.add(entry.message.id);
        }
        replyHeaded = true;
      } else if (entry.kind === "user" || entry.kind === "schedule") {
        replyHeaded = false;
      }
      if (!seenFirst) {
        first.add(entry.message.id);
        seenFirst = true;
      }
    }
    return { first, headed, any: seenFirst };
  }, [entries, placements, rowArtifacts]);
  const isFirst = (id: string) => layout.first.has(id);

  // A reply names an artifact in its sentence by id; the badge is drawn from
  // the conversation's artifacts, and from the ones the reply being written
  // has just made. An id the conversation never made stays words.
  const liveArtifacts = turn?.artifacts;
  const knownArtifacts = useMemo(() => {
    const known = new Map<string, Pick<AssistantArtifact, "id" | "kind" | "title">>();
    for (const artifact of liveArtifacts ?? []) known.set(artifact.id, artifact);
    for (const artifact of artifacts) known.set(artifact.id, artifact);
    return known;
  }, [artifacts, liveArtifacts]);
  const activeInline = workspaceOpen ? activeArtifactId : null;
  const renderArtifactLink = useCallback<ArtifactLinkRenderer>(
    (id, children) => {
      const artifact = knownArtifacts.get(id);
      return artifact ? (
        <DeskInlineArtifact artifact={artifact} active={activeInline === id} onOpen={openArtifact}>
          {children}
        </DeskInlineArtifact>
      ) : null;
    },
    [activeInline, knownArtifacts, openArtifact],
  );

  return (
    <AssistantAgentProvider agent={agent} delegates={agent?.delegates}>
      <ArtifactOpenerProvider onOpen={openArtifact}>
        <ArtifactLinkContext value={renderArtifactLink}>
          <DecisionFollowUpProvider value={model.followUpDecision}>
            <div className={cn("dk-stage", workspaceOpen && "dk-open")} onKeyDown={onKeyDown}>
              <div
                className={cn("dk-room", isActive && "dk-lit")}
                style={
                  dockHeight === null
                    ? undefined
                    : ({ "--dk-dock-h": `${dockHeight}px` } as CSSProperties)
                }
              >
                <div className="dk-scroll" ref={scrollRef} tabIndex={0}>
                  <div className="dk-grid dk-flow">
                    {hasOlder && (
                      <div ref={olderRef} className="dk-older" aria-live="polite">
                        {loadingOlder ? t("Reading earlier messages…") : ""}
                      </div>
                    )}
                    {entries.map((entry) => {
                      if (entry.kind === "compaction") {
                        return (
                          <DeskRow
                            key={entry.message.id}
                            kind="event"
                            className="dk-cmpd"
                            first={isFirst(entry.message.id)}
                          >
                            <DeskCompactMark message={entry.message} />
                          </DeskRow>
                        );
                      }
                      if (entry.kind === "user") {
                        return (
                          <DeskRow
                            key={entry.message.id}
                            kind="question"
                            first={isFirst(entry.message.id)}
                          >
                            <DeskQuestion
                              text={entry.message.content}
                              mentions={entry.message.mentions}
                              attachments={entry.message.attachments}
                            />
                          </DeskRow>
                        );
                      }
                      if (entry.kind === "schedule") {
                        const scheduleId = entry.message.scheduleId ?? "";
                        return (
                          <Fragment key={entry.message.id}>
                            <DeskRow kind="question" first={isFirst(entry.message.id)}>
                              <DeskQuestion text={entry.message.content} />
                            </DeskRow>
                            <DeskRow kind="event">
                              <DeskScheduleCard
                                schedule={
                                  schedulesQuery.data
                                    ? (scheduleById.get(scheduleId) ?? null)
                                    : undefined
                                }
                                now={clock}
                                timezone={timezone}
                                disabled={scheduleActions.busy}
                                onToggle={scheduleActions.toggle}
                                onDelete={scheduleActions.remove}
                              />
                            </DeskRow>
                          </Fragment>
                        );
                      }
                      if (entry.kind === "declined") {
                        return (
                          <DeskRow
                            key={entry.message.id}
                            kind="question"
                            first={isFirst(entry.message.id)}
                          >
                            <DeskQuestion
                              text={entry.message.content}
                              muted
                              tag={t("Not sent to the agent")}
                            />
                          </DeskRow>
                        );
                      }
                      if (entry.kind === "refusal") {
                        return (
                          <DeskRow
                            key={entry.message.id}
                            kind="event"
                            first={isFirst(entry.message.id)}
                          >
                            <DeskErrorCard
                              tone="neutral"
                              icon="shield"
                              title={t(
                                "This is outside what {0} can do",
                                agent?.name ?? t("the agent"),
                              )}
                              sub={entry.message.content}
                              actions={
                                refusedQuestion.has(entry.message.id) ? (
                                  <>
                                    <DeskErrorButton
                                      ink
                                      onClick={() => {
                                        model.onDraftChange(
                                          refusedQuestion.get(entry.message.id) ?? "",
                                        );
                                        setAgentPickerSignal((value) => value + 1);
                                      }}
                                    >
                                      {t("Ask another agent")}
                                    </DeskErrorButton>
                                    <DeskErrorButton
                                      onClick={() =>
                                        model.onDraftChange(
                                          refusedQuestion.get(entry.message.id) ?? "",
                                        )
                                      }
                                    >
                                      {t("Rephrase")}
                                    </DeskErrorButton>
                                  </>
                                ) : undefined
                              }
                            />
                          </DeskRow>
                        );
                      }
                      if (entry.kind === "decision") {
                        return null;
                      }
                      if (entry.kind === "handoff") {
                        const carried = entry.message.handoff;
                        return carried ? (
                          <DeskRow
                            key={entry.message.id}
                            kind="event"
                            first={isFirst(entry.message.id)}
                          >
                            <DeskHandoffCard
                              handoff={carried}
                              incoming={entry.message.kind === "HandoffBrief"}
                              time={turnTime(entry.message.createdAt, timezone, t)}
                              agentsById={desk.agentsById}
                            />
                          </DeskRow>
                        ) : null;
                      }
                      const failure = entry.message.failure;
                      if (failure) {
                        const arrived = arrivedBefore.get(entry.message.id) ?? "";
                        const question = questionBefore.get(entry.message.id) ?? "";
                        return (
                          <DeskRow
                            key={entry.message.id}
                            kind="event"
                            first={isFirst(entry.message.id)}
                          >
                            {failure.kind === "no_model" ? (
                              <DeskNoModelCard
                                providers={failure.providers}
                                onRetry={question !== "" ? () => ask(question) : undefined}
                                onCheckStatus={checkProviders}
                              />
                            ) : failure.kind === "stopped" || failure.kind === "interrupted" ? (
                              <DeskCutOffCard
                                arrived={arrived}
                                vendor={vendorOf(entry.message.providerId)}
                                stoppedByYou={failure.kind === "stopped"}
                                onRetry={question !== "" ? () => ask(question) : undefined}
                                onContinue={() => ask(t("Continue from where you stopped."))}
                              />
                            ) : (
                              <DeskErrorCard
                                tone="err"
                                compact
                                title={t("The reply didn't come through")}
                                sub={
                                  closingReason(entry.message.content) || t("Nothing was changed.")
                                }
                                actions={
                                  question !== "" ? (
                                    <DeskErrorButton ink onClick={() => ask(question)}>
                                      <DeskIcon name="replay" size={12} />
                                      {t("Try again")}
                                    </DeskErrorButton>
                                  ) : undefined
                                }
                              />
                            )}
                          </DeskRow>
                        );
                      }
                      const own = rowArtifacts.get(entry.message.id) ?? NO_ARTIFACTS;
                      if (!replyShows(entry, own.length)) {
                        return null;
                      }
                      const continued = !layout.headed.has(entry.message.id);
                      return (
                        <DeskRow
                          key={entry.message.id}
                          kind={continued ? "continued" : "reply"}
                          first={isFirst(entry.message.id)}
                          time={
                            continued ? undefined : turnTime(entry.message.createdAt, timezone, t)
                          }
                        >
                          <DeskReply
                            entry={entry}
                            steps={replySteps.get(entry.message.id) ?? NO_STEPS}
                            threadArtifacts={artifacts}
                            artifacts={own}
                            latestUserSequence={model.latestUserSequence}
                            chapter={chapterOf(entry.message.id)}
                            closesTurn={lastOfTurn.has(entry.message.id)}
                            onTogglePin={togglePin}
                            onAnswer={model.answer}
                            onOpenArtifact={openArtifact}
                          />
                          {(poorlyRead.get(entry.message.id) ?? []).map((fileName) => (
                            <div key={fileName} className="dk-ec-after">
                              <DeskPoorlyReadCard
                                fileName={fileName}
                                onUploadClearer={
                                  composerLock ? undefined : () => setFilePickerSignal((n) => n + 1)
                                }
                                onUseWhatWasRead={
                                  composerLock
                                    ? undefined
                                    : () =>
                                        ask(
                                          t(
                                            "Go ahead with what you could read from {0}.",
                                            fileName,
                                          ),
                                        )
                                }
                              />
                            </div>
                          ))}
                          {(writeResults.get(entry.message.id) ?? []).map((proposal) => (
                            <div key={proposal.id} className="dk-ec-after">
                              <DeskWriteResultCard proposal={proposal} onAsk={ask} />
                            </div>
                          ))}
                          {entry.message.fallbackFrom && (
                            <DeskFallbackLine
                              fromVendor={vendorOf(entry.message.fallbackFrom.providerId) ?? ""}
                              fromModel={
                                entry.message.fallbackFrom.model || entry.message.fallbackFrom.name
                              }
                              answeredVendor={vendorOf(entry.message.providerId) ?? ""}
                              answeredModel={entry.message.model}
                            />
                          )}
                          {entry.message.truncated && (
                            <DeskCutOffCard
                              arrived={withoutCutNote(entry.message.content)}
                              vendor={vendorOf(entry.message.providerId)}
                              onRetry={retryFor(entry.message.id)}
                              onContinue={() => ask(t("Continue from where you stopped."))}
                            />
                          )}
                        </DeskRow>
                      );
                    })}
                    {turn &&
                      !turn.followUp &&
                      turn.userContent !== "" &&
                      turn.rateLimited === null && (
                        <DeskRow kind="question" first={!layout.any}>
                          <DeskQuestion
                            text={turn.userContent}
                            attachments={turn.attachments}
                            mentions={turn.mentions}
                            muted={turn.status === "refused"}
                            tag={turn.status === "refused" ? t("Not sent to the agent") : undefined}
                          />
                        </DeskRow>
                      )}
                    {turn && live !== "" && (
                      <DeskRow
                        kind="reply"
                        first={!layout.any}
                        time={turnTime(Math.floor(turn.startedAt / 1000), timezone, t)}
                      >
                        <DeskStreamingReply
                          text={withArtifactRefs(live, keptWhileWriting(turn.artifacts))}
                          usedMemoryIds={turn.usedMemoryIds}
                        />
                      </DeskRow>
                    )}
                    {turn?.status === "error" &&
                      turn.limit === null &&
                      turn.rateLimited === null && (
                        <DeskRow kind="event" first={!layout.any}>
                          {turn.failedProviders.length > 0 ? (
                            <DeskNoModelCard
                              providers={turn.failedProviders}
                              onRetry={model.retry ? () => void model.retry?.() : undefined}
                              onCheckStatus={checkProviders}
                            />
                          ) : turn.stopped || live !== "" ? (
                            <DeskCutOffCard
                              arrived={live}
                              stoppedByYou={turn.stopped}
                              onRetry={model.retry ? () => void model.retry?.() : undefined}
                              onContinue={() => ask(t("Continue from where you stopped."))}
                            />
                          ) : (
                            <DeskErrorCard
                              tone="err"
                              compact
                              title={t("The reply didn't come through")}
                              sub={turn.error ?? t("Nothing was changed.")}
                              actions={
                                <>
                                  {model.retry && (
                                    <DeskErrorButton ink onClick={() => void model.retry?.()}>
                                      <DeskIcon name="replay" size={12} />
                                      {t("Try again")}
                                    </DeskErrorButton>
                                  )}
                                  <DeskErrorButton onClick={() => void model.dismiss()}>
                                    {t("Dismiss")}
                                  </DeskErrorButton>
                                </>
                              }
                            />
                          )}
                        </DeskRow>
                      )}
                    {/* The question that opened the conversation goes once the
                        thread's history has loaded; it shows from the first
                        frame, as it does when sent from here. */}
                    {!waitingOpening &&
                      !turn &&
                      openingSchedule === null &&
                      opening.openingQuestion && (
                        <DeskRow kind="question" first={entries.length === 0}>
                          <DeskQuestion
                            text={opening.openingQuestion}
                            mentions={handoff?.mentions}
                          />
                        </DeskRow>
                      )}
                    {waitingOpening && (
                      <DeskRow kind="question" first={entries.length === 0}>
                        <DeskQuestion
                          text={waitingOpening}
                          attachments={waitingFiles}
                          mentions={handoff?.mentions}
                        />
                        <div className="dk-ec-queued dk-ec-queued-row">
                          <DeskIcon name="up" size={11} stroke={2.2} />
                          {t(
                            "{0, plural, one {Uploading # file} other {Uploading # files}}",
                            Math.max(waitingFiles.length, handoff?.files.length ?? 0),
                          )}
                        </div>
                      </DeskRow>
                    )}
                    {scheduling.pending !== null && (
                      <DeskRow kind="question" first={entries.length === 0}>
                        <DeskQuestion text={scheduling.pending} />
                      </DeskRow>
                    )}
                    {queued && (
                      <DeskRow kind="question" first={entries.length === 0}>
                        <DeskQuestion text={queued.content} />
                        <div className="dk-ec-queued dk-ec-queued-row">
                          <DeskIcon name="undo" size={11} stroke={2.2} />
                          {t("Waiting to send")}
                        </div>
                      </DeskRow>
                    )}
                  </div>
                </div>
                {!online && (
                  <div className="dk-ec-off dk-ec-offtop" role="status">
                    <span className="dk-ec-offd" />
                    {t("You're offline · messages will send when you reconnect")}
                  </div>
                )}
                <DeskDropOverlay show={drag.on} hot={drag.hot} count={drag.count} />
                {burst !== null && <DeskConfetti key={burst} seed={burst % 1000} />}
                <div
                  className={cn("dk-jump", jumping && "dk-show", isActive && jumping && "dk-live")}
                >
                  <button
                    type="button"
                    className="dk-jump-b"
                    onClick={jumpToLatest}
                    tabIndex={jumping ? 0 : -1}
                  >
                    {isActive && jumping ? (
                      <>
                        <span className="dk-jump-d" />
                        {t("Agent is replying")}
                      </>
                    ) : unread > 0 ? (
                      <>
                        <span className="dk-jump-n">{unread}</span>
                        {t("{0, plural, one {New reply} other {New replies}}", unread)}
                      </>
                    ) : (
                      t("Jump to latest")
                    )}
                    <svg
                      width="12"
                      height="12"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2.4"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      aria-hidden
                    >
                      <path d="M12 5v14M6 13l6 6 6-6" />
                    </svg>
                  </button>
                </div>
                <div className="dk-dock" ref={dockRef}>
                  <div className="dk-grid">
                    <div className="dk-g" />
                    <div className={cn("dk-c", attached && "dk-has-dec")}>
                      {thread.canContinue && <DeskFactsBar thread={thread} />}
                      {!pending && !holding && <DeskTermsNote />}
                      {pending && model.current === null && !holding && (
                        <button
                          type="button"
                          className="dk-bt dk-sm dk-dock-pill"
                          onClick={model.resumeAll}
                        >
                          {t(
                            "{0, plural, one {# change waits on you} other {# changes wait on you}}",
                            model.queue.length,
                          )}
                        </button>
                      )}
                      {model.block === "full" && !composerLock && (
                        <div className="dk-ec-dockcard">
                          <DeskErrorCard
                            tone="info"
                            icon="info"
                            compact
                            title={t("This conversation is too long for {0}", currentModelName)}
                            sub={t(
                              "Start a new conversation with {0} to keep going. This one stays here to read.",
                              agent?.name ?? t("the agent"),
                            )}
                            actions={
                              <>
                                {onStartNew && (
                                  <DeskErrorButton ink onClick={onStartNew}>
                                    {t("Continue in a new conversation")}
                                  </DeskErrorButton>
                                )}
                                {model.providers.length > 1 && (
                                  <DeskErrorButton
                                    onClick={() => setPickerSignal((value) => value + 1)}
                                  >
                                    {t("Switch model")}
                                  </DeskErrorButton>
                                )}
                              </>
                            }
                          />
                        </div>
                      )}
                      {budgetQuery.data && !lock && <DeskUsageMeter budget={budgetQuery.data} />}
                      {undo.state.phase === "waiting" && (
                        <DeskUndoBar
                          key={undo.state.window.key}
                          held={undo.state.window}
                          left={undo.left}
                          busy={undo.busy}
                          onUndo={undo.undo}
                          onNow={undo.commitNow}
                        />
                      )}
                      {undo.state.phase === "committed" && (
                        <DeskApprovedCard held={undo.state.window} />
                      )}
                      {showCard && model.current && (
                        <DeskApprovalCard
                          key={model.current.entry.key}
                          threadId={thread.id}
                          entry={model.current.entry}
                          approveRef={approveRef}
                          onReview={reviewEntry}
                          onDefer={model.deferAll}
                          onDecided={model.decided}
                          undo={undo}
                          onAsk={ask}
                        />
                      )}
                      <DeskComposer
                        value={model.draft}
                        onChange={model.onDraftChange}
                        onSend={(content, payload) => {
                          if (isScheduleRequest(content)) {
                            // Kept as a schedule, not asked: its card says when
                            // it runs. Refused, the words go back in the box.
                            createSchedule(content, model.onDraftChange);
                            composerContext.clear();
                            attachments.clear();
                            return;
                          }
                          if (!online) {
                            setQueued({ content, payload });
                          } else {
                            void model.send(content, undefined, model.providerId, payload);
                          }
                          composerContext.clear();
                          attachments.clear();
                        }}
                        onStop={model.stop}
                        agent={agent}
                        onAgentChange={(next) => {
                          if (next.id !== agent?.id) {
                            desk.start(next.id, model.draft.trim() || undefined);
                            model.onDraftChange("");
                          }
                        }}
                        recentAgentIds={askable.recency.ids}
                        agentLastUsedAt={askable.recency.lastUsedAt}
                        busy={isActive}
                        status={
                          rateLimited !== null
                            ? {
                                text: t("You're sending messages quickly · send again in a moment"),
                                pose: "retry",
                                countdown: rateLimited,
                              }
                            : status
                        }
                        wait={rateLimited ?? 0}
                        agentPickerSignal={agentPickerSignal}
                        filePickerSignal={filePickerSignal}
                        placeholder={
                          showCard
                            ? t("Reply, or ask about this change…")
                            : lastIsRefusal
                              ? t("Rephrase, or ask another agent…")
                              : undefined
                        }
                        lock={lock}
                        note={composerLock?.note}
                        attachments={attachments}
                        scans={scans}
                        mentions={composerContext.mentions}
                        onMentionsChange={composerContext.setMentions}
                        suggestions={model.suggestions}
                        drag={drag}
                        extras={
                          <DeskPageChip
                            page={deskPage.page}
                            share={deskPage.share}
                            onShareChange={deskPage.setShare}
                            onExplain={explainPage}
                          />
                        }
                        model={
                          <DeskModelPicker
                            options={model.providers}
                            value={model.providerId}
                            onChange={model.setProviderId}
                            hasReplies={replies > 0}
                            disabled={isActive}
                            openSignal={pickerSignal}
                          />
                        }
                        meter={
                          <DeskContextMeter
                            usage={compaction.usage}
                            auto={compaction.auto}
                            onAutoChange={(on) => void compaction.setAuto(on)}
                            onCompact={() => void compaction.compact()}
                            compacting={compaction.compacting !== null}
                            budget={budgetQuery.data}
                          />
                        }
                        compacting={
                          compaction.compacting && {
                            auto: compaction.compacting.auto,
                            before: compaction.compacting.before,
                            after: compaction.compacting.after,
                            window: meterView(compaction.usage).window,
                          }
                        }
                        onCancelCompact={compaction.cancel}
                        onCompact={() => void compaction.compact()}
                      />
                      <div className="dk-hint">
                        {showCard ? (
                          <span>
                            {t("Press")} <span className="dk-kbd">⌘↵</span> {t("to approve")}
                          </span>
                        ) : (
                          <span>
                            {t(
                              "Desk can make mistakes. Check important details before you act on them.",
                            )}
                          </span>
                        )}
                      </div>
                    </div>
                    <div className="dk-m" />
                  </div>
                </div>
              </div>
              <aside className="dk-sheet" aria-label={t("Artifacts")}>
                {workspaceOpen && (
                  <DeskWorkspace
                    key={thread.id}
                    threadId={thread.id}
                    liveArtifacts={desk.liveArtifacts}
                    pendingLookups={desk.pendingLookups}
                    onClose={() => setWorkspaceOpen(false)}
                  />
                )}
              </aside>
            </div>
          </DecisionFollowUpProvider>
        </ArtifactLinkContext>
      </ArtifactOpenerProvider>
    </AssistantAgentProvider>
  );
}
