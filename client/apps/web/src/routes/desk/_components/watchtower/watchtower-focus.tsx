import { useApiMutation } from "@/hooks/use-api-mutation";
import { conversationPath } from "@/lib/conversation-path";
import {
  dismissWatchtowerItem,
  handOffWatchtowerItem,
  markWatchtowerSeen,
  snoozeWatchtowerItem,
  type WatchtowerItem,
} from "@/lib/graphql/watchtower";
import { queries } from "@/lib/queries";
import { formatTimeAgo } from "@/lib/time-utils";
import { apiService } from "@/services/api";
import { useAssistantStore } from "@/stores/assistant-store";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useNavigate } from "react-router";
import { toast } from "sonner";
import { DeskAgentTile } from "../desk-agent-tile";
import { DeskIcon, type DeskIconName } from "../desk-icons";
import { useDesk } from "../desk-layout";
import {
  clockText,
  formatSpan,
  inFocusOrder,
  laneOf,
  LANES,
  minutesLeft,
  runOpen,
  runStage,
  runSteps,
  type Lane,
} from "./lanes";

/** The most of the tower one visit reads; it is ordered by deadline here, not by the server. */
const FOCUS_PAGE = 200;
/** How long a card takes to leave once it is cleared. */
const LEAVE_MS = 300;

const nowInSeconds = () => Math.floor(Date.now() / 1000);

const KIND_ICON: Partial<Record<WatchtowerItem["sourceKind"], DeskIconName>> = {
  AgentRunFailed: "alert",
  AgentException: "alert",
  AgentProposal: "inbox",
  AgentPlan: "inbox",
  ServiceFailure: "headset",
  WorkerCredential: "lock",
  EDIInboundQuarantined: "file",
  MoveCoverage: "truck",
  TelematicsStopVisit: "truck",
  DetentionOccurrence: "receipt",
  CarrierIntelEvent: "shield",
  BillingException: "receipt",
  WeatherAlert: "compass",
  InboundMessage: "chat",
  HOSViolation: "route",
};

function laneTitle(lane: Lane, t: TranslateFn): string {
  switch (lane) {
    case "now":
      return t("Act now");
    case "today":
      return t("Today");
    case "later":
      return t("Coming up");
    default:
      return t("For your information");
  }
}

function runLine(run: NonNullable<WatchtowerItem["activeRun"]>, t: TranslateFn): string {
  const name = run.definition?.name ?? t("An agent");
  switch (runStage(run.status)) {
    case "needs":
      return t("{0} needs you", name);
    case "done":
      return t("{0} finished", name);
    case "failed":
      return t("{0} stopped", name);
    default:
      return t("{0} is on it", name);
  }
}

/** Tomorrow at 8 in the morning where the person is, in Unix seconds. */
function tomorrowMorning(now: number): number {
  const at = new Date(now * 1000);
  at.setDate(at.getDate() + 1);
  at.setHours(8, 0, 0, 0);
  return Math.floor(at.getTime() / 1000);
}

/**
 * The agent's mark inside a ring that fills as its run moves through its
 * stages, colored by where it stands: working, waiting on the person, done.
 */
function AgentRing({
  run,
  size = 22,
}: {
  run: NonNullable<WatchtowerItem["activeRun"]>;
  size?: number;
}) {
  const t = useT();
  const stage = runStage(run.status);
  const { steps, reached } = runSteps(run.status, t);
  const share = stage === "done" ? 1 : Math.min(1, (reached + 0.5) / steps.length);
  const radius = size / 2 + 2.5;
  const box = size + 8;
  const length = 2 * Math.PI * radius;
  return (
    <span
      className={cn("dk-wt-ar", `dk-st-${stage === "failed" ? "needs" : stage}`)}
      style={{ width: size, height: size }}
    >
      <DeskAgentTile agent={run.definition ?? null} size="xs" />
      <svg width={box} height={box} viewBox={`0 0 ${box} ${box}`} aria-hidden>
        <circle
          cx={box / 2}
          cy={box / 2}
          r={radius}
          fill="none"
          stroke="currentColor"
          strokeOpacity=".18"
          strokeWidth="1.6"
        />
        <circle
          cx={box / 2}
          cy={box / 2}
          r={radius}
          fill="none"
          stroke="currentColor"
          strokeWidth="1.6"
          strokeLinecap="round"
          strokeDasharray={length}
          strokeDashoffset={length * (1 - share)}
          transform={`rotate(-90 ${box / 2} ${box / 2})`}
          style={{ transition: "stroke-dashoffset 600ms" }}
        />
      </svg>
    </span>
  );
}

/**
 * The watchtower as a queue to work through: everything the system noticed,
 * laned by how long is left to head it off, one item in front of the person
 * at a time. An item an agent is already working on says so and how far it
 * has got; one nobody has taken names an agent that could.
 *
 * Done takes the item off the tower for everyone; snooze puts it aside for
 * this person until later; hand off gives it to an agent.
 */
export function WatchtowerFocus() {
  const t = useT();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { agents } = useDesk();
  const [now, setNow] = useState(nowInSeconds);
  useEffect(() => {
    const timer = window.setInterval(() => setNow(nowInSeconds()), 30_000);
    return () => window.clearInterval(timer);
  }, []);

  const filter = useMemo(() => ({ unresolvedOnly: true, first: FOCUS_PAGE }), []);
  const feedQuery = useQuery({ ...queries.watchtower.feed(filter), refetchInterval: 30_000 });
  const [gone, setGone] = useState<ReadonlySet<string>>(new Set());
  const [cleared, setCleared] = useState(0);
  const items = useMemo(
    () =>
      inFocusOrder(
        (feedQuery.data?.items ?? []).filter((item) => !gone.has(item.id)),
        now,
      ),
    [feedQuery.data?.items, gone, now],
  );

  const [currentId, setCurrentId] = useState<string | null>(null);
  const index = Math.max(
    0,
    items.findIndex((item) => item.id === currentId),
  );
  const current = items[index] ?? null;
  const [leaving, setLeaving] = useState<"ok" | "no" | null>(null);
  const [menu, setMenu] = useState(false);

  // Opening the tower is looking at it: the unseen line moves to now.
  useEffect(() => {
    void markWatchtowerSeen().then((counts) => {
      queryClient.setQueryData(queries.watchtower.counts().queryKey, counts);
    });
  }, [queryClient]);

  const refresh = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queries.watchtower._def }),
      queryClient.invalidateQueries({ queryKey: queries.attention._def }),
    ]);
  }, [queryClient]);

  const go = useCallback(
    (step: number) => {
      if (items.length === 0) return;
      setMenu(false);
      setCurrentId(items[(index + step + items.length) % items.length].id);
    },
    [index, items],
  );

  /** Takes the card off the tower, then reads it again. */
  const leave = useCallback(
    (id: string, how: "ok" | "no") => {
      setLeaving(how);
      setMenu(false);
      window.setTimeout(() => {
        const rest = items.filter((item) => item.id !== id);
        setGone((previous) => new Set(previous).add(id));
        setCleared((count) => count + 1);
        setLeaving(null);
        setCurrentId(rest[Math.min(index, rest.length - 1)]?.id ?? null);
        void refresh();
      }, LEAVE_MS);
    },
    [index, items, refresh],
  );

  const doneMutation = useApiMutation({
    mutationFn: (item: WatchtowerItem) => dismissWatchtowerItem(item.id),
    onSuccess: (item) => leave(item.id, "ok"),
    resourceName: "Watchtower item",
  });

  const snoozeMutation = useApiMutation({
    mutationFn: ({ item, until }: { item: WatchtowerItem; until: number }) =>
      snoozeWatchtowerItem(item.id, until),
    onSuccess: (item, { until }) => {
      toast.success(
        t(
          "Snoozed until {0}",
          new Date(until * 1000).toLocaleString(undefined, {
            weekday: "short",
            hour: "numeric",
            minute: "2-digit",
          }),
        ),
      );
      leave(item.id, "no");
    },
    resourceName: "Watchtower item",
  });

  const handOffMutation = useApiMutation({
    mutationFn: ({ item, agentId }: { item: WatchtowerItem; agentId?: string }) =>
      handOffWatchtowerItem(item.id, agentId),
    onSuccess: async (result) => {
      setMenu(false);
      if (result.runId !== null) {
        toast.success(t("An agent is working on it"));
      } else if (result.subscribers.length > 0) {
        toast.success(t("Handed to {0}", result.subscribers.map((agent) => agent.name).join(", ")));
      } else {
        toast.info(t("No agent takes this on its own yet"), {
          description: t("Pick one to hand it to."),
        });
        setMenu(true);
      }
      await refresh();
    },
    resourceName: "Watchtower item",
  });

  // Asking about an item opens a conversation on its record, with the agent
  // the person last talked to, so it starts with the subject in hand.
  const lastAgentId = useAssistantStore((state) => state.lastAgentId);
  const askAgent = agents.find((agent) => agent.id === lastAgentId) ?? agents[0] ?? null;
  const askMutation = useApiMutation({
    mutationFn: (item: WatchtowerItem) => {
      if (askAgent === null) {
        throw new Error(t("No agents are available to ask"));
      }
      return apiService.assistantService.startThread(askAgent.id, {
        origin: "Watchtower",
        subjectType: item.subjectType ?? undefined,
        subjectId: item.subjectId ?? undefined,
      });
    },
    onSuccess: (thread) => navigate(conversationPath(thread.id)),
    resourceName: "Conversation",
  });

  const busy =
    doneMutation.isPending ||
    snoozeMutation.isPending ||
    handOffMutation.isPending ||
    leaving !== null;
  const run = current?.activeRun ?? null;
  const working = run !== null && runOpen(run.status);
  const canHandOff = current !== null && current.subjectId && !working;

  const snoozeUntil = current
    ? laneOf(current, now) === "now"
      ? now + 3600
      : tomorrowMorning(now)
    : 0;
  const done = useCallback(() => {
    if (current && !busy) doneMutation.mutate(current);
  }, [busy, current, doneMutation]);
  const snooze = useCallback(() => {
    if (current && !busy) snoozeMutation.mutate({ item: current, until: snoozeUntil });
  }, [busy, current, snoozeMutation, snoozeUntil]);
  const handOff = useCallback(
    (agentId?: string) => {
      if (!current || busy || !canHandOff) return;
      handOffMutation.mutate({
        item: current,
        agentId: agentId ?? current.suggestedAgent?.id ?? undefined,
      });
    },
    [busy, canHandOff, current, handOffMutation],
  );

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      if (target?.closest("input,textarea,select,[contenteditable=true],[role=dialog]")) return;
      if (event.metaKey || event.ctrlKey || event.altKey) return;
      const key = event.key.toLowerCase();
      if (key === "j" || event.key === "ArrowDown") {
        event.preventDefault();
        go(1);
      } else if (key === "k" || event.key === "ArrowUp") {
        event.preventDefault();
        go(-1);
      } else if (key === "e") {
        event.preventDefault();
        done();
      } else if (key === "s") {
        event.preventDefault();
        snooze();
      } else if (key === "h") {
        event.preventDefault();
        handOff();
      } else if (event.key === "Escape") {
        setMenu(false);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [done, go, handOff, snooze]);

  const onIt = items.filter((item) => item.activeRun && runOpen(item.activeRun.status));
  const total = items.length + cleared;
  const clock = current ? clockText(current, now, t) : null;
  const lane = current ? laneOf(current, now) : null;
  const left = current ? minutesLeft(current, now) : null;

  return (
    <div className="dk-w3">
      <aside className="dk-w3-q" aria-label={t("Watchtower")}>
        <div className="dk-dc2-qh">
          <b>{t("Watchtower")}</b>
          <span>{t("{0} of {1} cleared", cleared, total)}</span>
        </div>
        <div className="dk-dc2-prog">
          <i style={{ width: `${total ? (cleared / total) * 100 : 0}%` }} />
        </div>
        {onIt.length > 0 && (
          <div className="dk-w3-on">
            <span className="dk-w3-onl">{t("On it")}</span>
            {onIt.slice(0, 6).map((item) => (
              <button
                key={item.id}
                type="button"
                title={`${item.activeRun?.definition?.name ?? ""} · ${item.title}`}
                onClick={() => setCurrentId(item.id)}
              >
                {item.activeRun && <AgentRing run={item.activeRun} size={18} />}
              </button>
            ))}
            <span className="dk-w3-onn">{t("{0} working", onIt.length)}</span>
          </div>
        )}
        <div className="dk-dc2-ql">
          {LANES.map((key) => {
            const laneItems = items.filter((item) => laneOf(item, now) === key);
            if (laneItems.length === 0) return null;
            return (
              <div key={key} className={cn("dk-dc2-g", `dk-w3-l-${key}`)}>
                <div className="dk-dc2-gh">
                  <b className="dk-w3-lt">{laneTitle(key, t)}</b>
                  <i>{laneItems.length}</i>
                </div>
                {laneItems.map((item) => {
                  const itemLeft = minutesLeft(item, now);
                  return (
                    <button
                      key={item.id}
                      type="button"
                      className={cn("dk-dc2-qi dk-w3-qi", current?.id === item.id && "dk-on")}
                      aria-current={current?.id === item.id}
                      onClick={() => {
                        setMenu(false);
                        setCurrentId(item.id);
                      }}
                    >
                      <span className={cn("dk-w3-ic", `dk-s-${item.severity}`)}>
                        {item.activeRun ? (
                          <DeskAgentTile agent={item.activeRun.definition ?? null} size="xs" />
                        ) : (
                          <DeskIcon name={KIND_ICON[item.sourceKind] ?? "info"} size={12} />
                        )}
                      </span>
                      <span>
                        <b>{item.title}</b>
                        <em>{item.activeRun ? runLine(item.activeRun, t) : item.kindLabel}</em>
                      </span>
                      {itemLeft !== null && itemLeft < 1440 ? (
                        <span className={cn("dk-w3-due", itemLeft <= 60 && "dk-hot")}>
                          {itemLeft <= 0
                            ? t("{0} over", formatSpan(itemLeft, t))
                            : formatSpan(itemLeft, t)}
                        </span>
                      ) : !item.seen ? (
                        <span className="dk-dc2-new" aria-label={t("New")} />
                      ) : null}
                    </button>
                  );
                })}
              </div>
            );
          })}
          {feedQuery.data?.hasNextPage && (
            <div className="dk-dc2-qe">{t("Showing the newest {0}.", FOCUS_PAGE)}</div>
          )}
          {!feedQuery.isLoading && items.length === 0 && (
            <div className="dk-dc2-qe">{t("All clear")}</div>
          )}
        </div>
      </aside>

      <main className="dk-dc2-main">
        {feedQuery.isLoading ? (
          <div className="dk-dc2-clear">
            <span className="dk-dc2-qe">{t("Reading the tower…")}</span>
          </div>
        ) : current ? (
          <div
            key={current.id}
            className={cn("dk-dc2-card dk-w3-card", leaving && `dk-out-${leaving}`)}
          >
            {clock && left !== null && (
              <div className={cn("dk-w3-clock", `dk-l-${lane}`)}>
                <span className="dk-w3-cr">
                  <svg width="44" height="44" viewBox="0 0 44 44" aria-hidden>
                    <circle
                      cx="22"
                      cy="22"
                      r="19"
                      fill="none"
                      stroke="currentColor"
                      strokeOpacity=".18"
                      strokeWidth="3"
                    />
                    <circle
                      cx="22"
                      cy="22"
                      r="19"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="3"
                      strokeLinecap="round"
                      strokeDasharray={119.4}
                      strokeDashoffset={119.4 * Math.min(1, Math.max(0, left) / 240)}
                      transform="rotate(-90 22 22)"
                      style={{ transition: "stroke-dashoffset 1s linear" }}
                    />
                  </svg>
                </span>
                <span>
                  <b>{clock.time}</b>
                  <em>{clock.label}</em>
                </span>
              </div>
            )}
            <div className="dk-w3-kind">
              <span className={cn("dk-wt-sev", `dk-s-${current.severity}`)} />
              {current.kindLabel}
              <span> · {formatTimeAgo(current.occurredAt * 1000)}</span>
            </div>
            <h1>{current.title}</h1>
            {current.summary !== "" && <p className="dk-dc2-why">{current.summary}</p>}

            {current.path !== "" && (
              <Link className="dk-w3-rec" to={current.path}>
                <DeskIcon name={KIND_ICON[current.sourceKind] ?? "info"} size={14} />
                <span>
                  <em>{current.kindLabel}</em>
                  <b>{t("Open the record")}</b>
                </span>
                <span className="dk-w3-recf" />
                <DeskIcon name="ext" size={12} />
              </Link>
            )}

            {run ? (
              <RunPanel run={run} onReview={() => void navigate("/desk/decisions")} />
            ) : current.suggestedAgent ? (
              <div className="dk-w3-sug">
                <DeskAgentTile agent={current.suggestedAgent} size="sm" />
                <span>
                  <b>{t("{0} can take this", current.suggestedAgent.name)}</b>
                  {current.suggestedAgent.description && (
                    <em>{current.suggestedAgent.description}</em>
                  )}
                </span>
              </div>
            ) : null}
          </div>
        ) : (
          <div className="dk-dc2-clear">
            <span className="dk-dc2-cic">
              <DeskIcon name="check" size={22} stroke={2.2} />
            </span>
            <b>{t("All clear")}</b>
            <span>
              {cleared > 0
                ? t(
                    "{0, plural, one {You cleared # item.} other {You cleared # items.}} New signals show up here the moment they happen.",
                    cleared,
                  )
                : t("New signals show up here the moment they happen.")}
            </span>
          </div>
        )}

        {current && (
          <div className="dk-dc2-bar">
            <button
              type="button"
              className="dk-dc2-nav"
              onClick={() => go(-1)}
              title={t("Previous (K)")}
              aria-label={t("Previous")}
            >
              <DeskIcon name="chevR" size={12} stroke={2.4} />
            </button>
            <button
              type="button"
              className="dk-dc2-nav dk-dn"
              onClick={() => go(1)}
              title={t("Next (J)")}
              aria-label={t("Next")}
            >
              <DeskIcon name="chevR" size={12} stroke={2.4} />
            </button>
            {askAgent && (
              <button
                type="button"
                className="dk-dc2-nav dk-w3-ask"
                style={{ marginLeft: 6 }}
                onClick={() => askMutation.mutate(current)}
                disabled={askMutation.isPending}
                title={t("Ask about this")}
              >
                <DeskIcon name="chat" size={13} />
                <span>{t("Ask about this")}</span>
              </button>
            )}
            <span style={{ flex: 1 }} />
            <button
              type="button"
              className="dk-dc2-btn"
              onClick={snooze}
              disabled={busy}
              title={t(
                "Until {0}",
                new Date(snoozeUntil * 1000).toLocaleString(undefined, {
                  weekday: "short",
                  hour: "numeric",
                  minute: "2-digit",
                }),
              )}
            >
              {t("Snooze")}
              <span className="dk-kbd">S</span>
            </button>
            <button type="button" className="dk-dc2-btn" onClick={done} disabled={busy}>
              {t("Done")}
              <span className="dk-kbd">E</span>
            </button>
            {canHandOff && (
              <span className="dk-wt-hm">
                <button
                  type="button"
                  className="dk-dc2-btn dk-ink"
                  onClick={() => handOff()}
                  disabled={busy}
                >
                  {current.suggestedAgent
                    ? t("Hand to {0}", current.suggestedAgent.name)
                    : t("Hand off")}
                  <span className="dk-kbd">H</span>
                </button>
                <button
                  type="button"
                  className="dk-dc2-btn dk-ink dk-more"
                  aria-label={t("Pick an agent")}
                  aria-expanded={menu}
                  onClick={() => setMenu((value) => !value)}
                >
                  <svg
                    width="10"
                    height="10"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2.6"
                    strokeLinecap="round"
                    aria-hidden
                  >
                    <path d="M6 15l6-6 6 6" />
                  </svg>
                </button>
                {menu && (
                  <div className="dk-wt-hp" role="menu">
                    {agents
                      .filter((agent) => agent.id !== current.suggestedAgent?.id)
                      .slice(0, 8)
                      .map((agent) => (
                        <button
                          key={agent.id}
                          type="button"
                          role="menuitem"
                          onClick={() => handOff(agent.id)}
                        >
                          <DeskAgentTile agent={agent} size="sm" />
                          <span>
                            <b>{agent.name}</b>
                            {agent.description && <em>{agent.description}</em>}
                          </span>
                        </button>
                      ))}
                  </div>
                )}
              </span>
            )}
          </div>
        )}
      </main>
    </div>
  );
}

function RunPanel({
  run,
  onReview,
}: {
  run: NonNullable<WatchtowerItem["activeRun"]>;
  onReview: () => void;
}) {
  const t = useT();
  const stage = runStage(run.status);
  const { steps, reached } = runSteps(run.status, t);
  const line =
    stage === "working"
      ? t("Step {0} of {1} · {2}", reached + 1, steps.length, steps[reached])
      : stage === "needs"
        ? t("Waiting for you to decide")
        : stage === "failed"
          ? t("Stopped with an error")
          : run.summary || t("Finished");
  return (
    <div className={cn("dk-w3-ag", `dk-st-${stage === "failed" ? "needs" : stage}`)}>
      <div className="dk-w3-agh">
        <AgentRing run={run} />
        <span>
          <b>{run.definition?.name ?? t("An agent")}</b>
          <em>{line}</em>
        </span>
        {stage === "needs" && (
          <button type="button" className="dk-dc2-btn dk-ink dk-sm" onClick={onReview}>
            {t("Review & approve")}
          </button>
        )}
      </div>
      <div className="dk-w3-agbar">
        {steps.map((step, index) => (
          <i
            key={step}
            title={step}
            className={cn(
              (index < reached || stage === "done") && "dk-d",
              index === reached && stage === "working" && "dk-n",
            )}
          />
        ))}
      </div>
    </div>
  );
}
