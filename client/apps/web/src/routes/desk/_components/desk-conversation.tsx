import { AssistantAgentProvider } from "@/components/agent-identity/agent-context";
import type { ApprovalEntry } from "@/components/assistant/approval-queue";
import { ArtifactOpenerProvider } from "@/components/assistant/artifact-opener";
import { DecisionFollowUpProvider } from "@/components/assistant/decision-follow-up";
import { useOpeningQuestion } from "@/components/assistant/use-opening-question";
import { useThreadModel } from "@/components/assistant/use-thread-model";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { useAssistantStore } from "@/stores/assistant-store";
import { useDeskStore } from "@/stores/desk-store";
import type { AssistantThread } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort, formatUnixTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useReducedMotion } from "motion/react";
import { useCallback, useEffect, useMemo, useRef, useState, type KeyboardEvent } from "react";
import { ArtifactsPane } from "./artifacts/artifacts-pane";
import { DeskComposer } from "./composer/desk-composer";
import {
  DeskApprovalCard,
  DeskApprovedCard,
  type ApprovedNote,
} from "./conversation/desk-approval-card";
import { DeskConfetti } from "./conversation/desk-confetti";
import {
  DeskArtifactBadges,
  DeskQuestion,
  DeskReply,
  DeskRow,
  DeskStreamingReply,
} from "./conversation/desk-turns";
import { composerStatus, streamingText } from "./conversation/turn-status";
import { useStickToBottom } from "./conversation/use-stick-to-bottom";
import { DeskErrorButton, DeskErrorCard } from "./desk-error-card";
import { DeskIcon } from "./desk-icons";
import { useDesk } from "./desk-layout";
import { DeskTermsNote } from "./desk-terms-note";

export type DeskConversationProps = {
  thread: AssistantThread;
  agent: AgentChoice | null;
  agentsUnavailable: boolean;
  onStartNew?: () => void;
};

/** How long the green "Approved" row holds before the composer has the room again. */
const APPROVED_HOLD_MS = 1100;
/** How long a confetti burst lasts. */
const CONFETTI_MS = 5400;

function turnTime(at: number, timezone: string, t: TranslateFn): string {
  const today = new Date().toLocaleDateString("en-CA", { timeZone: timezone });
  const day = new Date(at * 1000).toLocaleDateString("en-CA", { timeZone: timezone });
  if (day === today) {
    return formatUnixTime(at, { timezone }) || t("Now");
  }

  return formatUnixDateTimeShort(at, { timezone });
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
  const timezone = useAuthStore((state) => state.user?.timezone) || "UTC";
  const opening = useOpeningQuestion(thread.id);
  const artifactsQuery = useQuery(queries.assistant.artifacts(thread.id));
  const artifacts = useMemo(() => artifactsQuery.data?.results ?? [], [artifactsQuery.data]);
  const activeArtifactId = useDeskStore(
    (state) => state.activeArtifactByThread[thread.id] ?? null,
  );
  const chapters = useDeskStore((state) => state.chaptersByThread[thread.id]);
  const toggleChapter = useDeskStore((state) => state.toggleChapter);

  const setActiveThreadId = useAssistantStore((state) => state.setActiveThreadId);
  const openWidget = useAssistantStore((state) => state.openWidget);
  const carryConversation = useCallback(() => {
    setActiveThreadId(thread.id);
    openWidget();
  }, [openWidget, setActiveThreadId, thread.id]);

  const model = useThreadModel({
    thread,
    agent,
    agentsUnavailable,
    artifacts,
    onLiveArtifacts: desk.noteLiveArtifacts,
    onWorkingChange: desk.setWorking,
    onNavigate: carryConversation,
    openingQuestion: opening.openingQuestion,
    onOpeningQuestionSent: opening.onOpeningQuestionSent,
  });
  const { entries, placements, turn, isActive } = model;

  const { setArtifactCount, openArtifact: openInDesk, setWorkspaceOpen, workspaceOpen } = desk;
  useEffect(() => setArtifactCount(artifacts.length), [artifacts.length, setArtifactCount]);

  const openArtifact = useCallback(
    (artifactId: string) => openInDesk(thread.id, artifactId),
    [openInDesk, thread.id],
  );

  const replies = entries.filter((entry) => entry.kind === "assistant").length;
  const ownMessages = entries.filter((entry) => entry.kind === "user").length + (turn ? 1 : 0);
  const { scrollRef, away, unread, jumpToLatest } = useStickToBottom({ replies, ownMessages });

  const [approved, setApproved] = useState<ApprovedNote | null>(null);
  const [burst, setBurst] = useState<number | null>(null);
  const approveRef = useRef<(() => void) | null>(null);
  useEffect(() => {
    if (!approved) {
      return;
    }
    const timer = window.setTimeout(() => setApproved(null), APPROVED_HOLD_MS);
    return () => window.clearTimeout(timer);
  }, [approved]);
  useEffect(() => {
    if (burst === null) {
      return;
    }
    const timer = window.setTimeout(() => setBurst(null), CONFETTI_MS);
    return () => window.clearTimeout(timer);
  }, [burst]);

  const onApproved = (note: ApprovedNote) => {
    setApproved(note);
    if (!reduceMotion) {
      setBurst(Date.now());
    }
  };

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

  const status = composerStatus(turn, t);
  const live = streamingText(turn);
  const chapterOf = (id: string) => (chapters ? chapters.indexOf(id) + 1 : 0);
  const showCard = model.showDock && model.current !== null && approved === null;
  const pending = model.queue.length > 0;
  const jumping = away || unread > 0;

  const lock =
    model.block === "read-only" ? (
      <>
        <DeskIcon name="lock" size={13} stroke={2} />
        <span>
          <b>{t("This conversation can no longer continue.")}</b> {t("You can still read it.")}
        </span>
      </>
    ) : model.block === "full" ? (
      <>
        <DeskIcon name="info" size={13} stroke={2} />
        <span>
          <b>{t("This conversation is full.")}</b> {t("Start a new one to continue.")}
        </span>
        {onStartNew && (
          <button type="button" className="dk-ec-link" onClick={onStartNew}>
            {t("Continue in a new conversation")}
          </button>
        )}
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
          <b>{t("This agent has been turned off.")}</b>{" "}
          {t("You can still read this conversation.")}
        </span>
      </>
    ) : null;

  let firstRow = true;
  const isFirst = () => {
    const value = firstRow;
    firstRow = false;
    return value;
  };

  return (
    <AssistantAgentProvider agent={agent} delegates={agent?.delegates}>
      <ArtifactOpenerProvider onOpen={openArtifact}>
        <DecisionFollowUpProvider value={model.followUpDecision}>
          <div className={cn("dk-stage", workspaceOpen && "dk-open")} onKeyDown={onKeyDown}>
            <div className={cn("dk-room", isActive && "dk-lit")}>
              <div className="dk-scroll" ref={scrollRef} tabIndex={0}>
                <div className="dk-grid dk-flow">
                  {entries.map((entry) => {
                    if (entry.kind === "user") {
                      return (
                        <DeskRow key={entry.message.id} kind="question" first={isFirst()}>
                          <DeskQuestion text={entry.message.content} />
                        </DeskRow>
                      );
                    }
                    if (entry.kind === "declined") {
                      return (
                        <DeskRow key={entry.message.id} kind="question" first={isFirst()}>
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
                        <DeskRow key={entry.message.id} kind="event" first={isFirst()}>
                          <DeskErrorCard
                            tone="neutral"
                            icon="shield"
                            title={t("This is outside what {0} can do", agent?.name ?? t("the agent"))}
                            sub={entry.message.content}
                          />
                        </DeskRow>
                      );
                    }
                    if (entry.kind === "decision") {
                      return null;
                    }
                    const own = model.artifactsByMessage.get(entry.message.id) ?? [];
                    if (!replyShows(entry, own.length)) {
                      return null;
                    }
                    const continued = placements.get(entry.message.id)?.continued ?? false;
                    return (
                      <DeskRow
                        key={entry.message.id}
                        kind={continued ? "continued" : "reply"}
                        first={isFirst()}
                        time={
                          continued ? undefined : turnTime(entry.message.createdAt, timezone, t)
                        }
                      >
                        <DeskReply
                          entry={entry}
                          artifacts={own}
                          activeArtifactId={workspaceOpen ? activeArtifactId : null}
                          latestUserSequence={model.latestUserSequence}
                          chapter={chapterOf(entry.message.id)}
                          onTogglePin={() => toggleChapter(thread.id, entry.message.id)}
                          onAnswer={model.answer}
                          onOpenArtifact={openArtifact}
                        />
                      </DeskRow>
                    );
                  })}
                  {turn && !turn.followUp && turn.userContent !== "" && (
                    <DeskRow kind="question" first={isFirst()}>
                      <DeskQuestion
                        text={turn.userContent}
                        muted={turn.status === "refused"}
                        tag={turn.status === "refused" ? t("Not sent to the agent") : undefined}
                      />
                    </DeskRow>
                  )}
                  {turn && live !== "" && (
                    <DeskRow
                      kind="reply"
                      first={isFirst()}
                      time={turnTime(Math.floor(turn.startedAt / 1000), timezone, t)}
                    >
                      <DeskStreamingReply text={live} />
                      <DeskArtifactBadges
                        artifacts={turn.artifacts}
                        activeId={workspaceOpen ? activeArtifactId : null}
                        onOpen={openArtifact}
                      />
                    </DeskRow>
                  )}
                  {turn?.status === "error" && (
                    <DeskRow kind="event" first={isFirst()}>
                      <DeskErrorCard
                        tone="err"
                        compact
                        title={
                          live !== "" ? t("Reply stopped partway") : t("The reply didn't come through")
                        }
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
                    </DeskRow>
                  )}
                </div>
              </div>
              {burst !== null && <DeskConfetti key={burst} seed={burst % 1000} />}
              <div className={cn("dk-jump", jumping && "dk-show", isActive && jumping && "dk-live")}>
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
              <div className="dk-dock">
                <div className="dk-grid">
                  <div className="dk-g" />
                  <div className="dk-c">
                    {!pending && approved === null && <DeskTermsNote />}
                    {approved && <DeskApprovedCard note={approved} />}
                    {showCard && model.current && (
                      <DeskApprovalCard
                        key={model.current.entry.key}
                        threadId={thread.id}
                        entry={model.current.entry}
                        approveRef={approveRef}
                        onReview={reviewEntry}
                        onDefer={model.deferAll}
                        onDecided={model.decided}
                        onApproved={onApproved}
                      />
                    )}
                    {pending && model.current === null && approved === null && (
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
                    <DeskComposer
                      value={model.draft}
                      onChange={model.onDraftChange}
                      onSend={(content) =>
                        void model.send(content, undefined, model.providerId, {
                          attachments: [],
                          mentions: [],
                        })
                      }
                      onStop={model.stop}
                      agent={agent}
                      busy={isActive}
                      status={status}
                      lock={lock}
                    />
                    <div className="dk-hint">
                      {showCard ? (
                        <span>
                          {t("Press")} <span className="dk-kbd">⌘↵</span> {t("to approve")}
                        </span>
                      ) : (
                        <span>
                          {t("Desk can make mistakes. Check important details before you act on them.")}
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="dk-m" />
                </div>
              </div>
            </div>
            <aside className="dk-sheet">
              {workspaceOpen && (
                <div className="dk-panel">
                  <ArtifactsPane
                    key={thread.id}
                    threadId={thread.id}
                    liveArtifacts={desk.liveArtifacts}
                    onClose={() => setWorkspaceOpen(false)}
                    className="h-full"
                  />
                </div>
              )}
            </aside>
          </div>
        </DecisionFollowUpProvider>
      </ArtifactOpenerProvider>
    </AssistantAgentProvider>
  );
}
