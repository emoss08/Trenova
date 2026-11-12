import { ItemPanel } from "./panels/item-panel";
import { AgentWorkBlock } from "./agent-work";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import {
  askAgentMessage,
  wouldFailReasons,
  type WouldFailReason,
} from "@/components/assistant/proposal-preview/preview-warnings";
import {
  ReviewBlocker,
  ReviewedChange,
  ReviewedPlan,
} from "@/components/assistant/proposal-preview/reviewed-change";
import { useProposalPreview } from "@/components/assistant/proposal-preview/use-proposal-preview";
import { RefreshCcw01Icon, Lock01Icon } from "@trenova/shared/components/icons";
import { presentProposal } from "@/components/assistant/proposal-presenters";
import type { DeskAttentionGroup, DeskAttentionItem } from "@/lib/graphql/desk-attention";
import { deskDecisionPath } from "@/lib/desk-paths";
import { CornerDownLeftIcon } from "@trenova/shared/components/icons";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatMinutesSpan, formatUnixTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { Link, useNavigate } from "react-router";
import { WatchtowerHandOff } from "../watchtower/hand-off-menu";
import { formatCountdown, formatFactTime, minutesLeft } from "../watchtower/lanes";
import type { WatchtowerActions } from "../watchtower/use-watchtower-actions";
import { casePath, START_CASE } from "./attention-record";
import { caseWorkOf } from "./case-work";
import { dueTone, recordContext } from "./attention-row";
import { DeclineDialog } from "./decline-dialog";
import { LifeBadge, itemLife } from "./life-badge";
import type { ItemDecision } from "./use-item-decision";

const NO_REASONS: WouldFailReason[] = [];

/**
 * Why the one change in front of the person would be refused if approved, or
 * nothing when it would run. Read from the same preview the card shows, so it
 * costs no second request. A plan says this step by step in its own preview.
 */
function useRefusal(decision: ItemDecision): WouldFailReason[] {
  const single = decision.reviewed && decision.planId === null && decision.proposals.length === 1;
  const preview = useProposalPreview({
    scope: "approver",
    id: decision.proposals[0]?.id ?? "",
    enabled: single,
  }).data;
  if (!single || !preview) {
    return NO_REASONS;
  }

  return preview.warnings.flatMap(wouldFailReasons);
}

/**
 * The change an agent asks to make, as one card: what it would do in a
 * sentence, why (when that is not what the page already said), the values it
 * would set, and two quiet facts under them: whether it can be taken back and
 * who proposed it. Nothing here is a table of things the reader cannot change.
 */
function DecisionCard({
  decision,
  agentName,
  raised,
  said,
  blocked,
}: {
  decision: ItemDecision;
  agentName: string | null;
  raised: string;
  /** What the page has already said above the card, so the card does not repeat it. */
  said: string;
  blocked: readonly WouldFailReason[];
}) {
  const t = useT();
  const proposal = decision.proposals[0] ?? null;

  if (decision.loading) {
    return (
      <section className="mt-6" aria-busy aria-label={t("Reading the change")}>
        <div className="flex flex-col gap-2.5 rounded-xl border border-dsk-b bg-dsk-raised px-4 py-4">
          <i className="h-4 w-2/3 ui-shimmer rounded-[4px]" />
          <i className="h-3.5 w-1/2 ui-shimmer rounded-[4px]" />
          <i className="h-3.5 w-1/3 ui-shimmer rounded-[4px]" />
        </div>
      </section>
    );
  }
  if (decision.failedToRead || proposal === null) {
    return (
      <p className="dk-w-quiet mt-6">{t("The change could not be read. Open it on Decisions.")}</p>
    );
  }

  const views = decision.proposals.map((step) => presentProposal(step));
  const lead = views[0]?.summary ?? views[0]?.title ?? "";
  const why = proposal.rationale.trim();
  const reversible = views.every((view) => view.reversible);
  const previewed = decision.phase === "idle" && decision.reviewed;

  return (
    <section className="mt-6" data-slot="decision-card">
      <div className="overflow-hidden rounded-xl border border-dsk-b bg-dsk-raised">
        <div className="flex flex-col gap-1.5 px-4 pt-3.5 pb-3.5">
          <span className="text-dsk-sm text-dsk-subtle">
            {agentName ? t("{0} would", agentName) : t("It would")}
          </span>
          {views.length > 1 ? (
            <ol className="flex flex-col gap-1 text-dsk-base text-dsk-fg">
              {views.map((view, index) => (
                <li key={decision.proposals[index]?.id ?? index} className="flex gap-2">
                  <span className="text-dsk-subtle tabular-nums">{index + 1}</span>
                  <span className="min-w-0">{view.summary ?? view.title}</span>
                </li>
              ))}
            </ol>
          ) : (
            <p className="text-dsk-base text-dsk-fg">{lead}</p>
          )}
          {why !== "" && why !== said.trim() && <p className="text-dsk-sm text-dsk-fg2">{why}</p>}
        </div>

        {previewed && (
          <div className="border-t border-dsk-b px-4 py-3.5">
            {decision.planId ? (
              <ReviewedPlan
                planId={decision.planId}
                onReport={decision.onReport}
                scope="approver"
              />
            ) : (
              <ReviewedChange
                proposal={proposal}
                onReport={decision.onReport}
                scope="approver"
                titled={false}
              />
            )}
          </div>
        )}

        <div className="flex flex-wrap items-center gap-x-4 gap-y-1 border-t border-dsk-b px-4 py-2.5 text-dsk-sm text-dsk-subtle">
          {blocked.length === 0 && (
            <span className="inline-flex items-center gap-1.5">
              {reversible ? (
                <RefreshCcw01Icon aria-hidden className="size-3.5" />
              ) : (
                <Lock01Icon aria-hidden className="size-3.5" />
              )}
              {reversible ? t("Can be undone after it runs") : t("Can't be undone once it runs")}
            </span>
          )}
          <span className="ml-auto">{t("Proposed {0}", raised)}</span>
        </div>
      </div>
      {/* The Decisions lens reads it on the sheet of its kind, beside the others like it. */}
      {decision.planId === null && (
        <p className="mt-2.5 px-1 text-dsk-sm">
          <Link
            className="dk-w-alnk ui-focus-ring"
            to={deskDecisionPath({ proposal: proposal.id })}
          >
            {t("Approve several at once in Decisions")}
          </Link>
        </p>
      )}
    </section>
  );
}

/** Kbd-style hint inside a button, inverted on an ink button. */
function Key({ k, inv = false }: { k: ReactNode; inv?: boolean }) {
  return <span className={cn("dk-w-kbd", inv && "dk-inv")}>{k}</span>;
}

/**
 * One item of Needs you, read in full beside the list: what it is about and
 * how long is left; the condition the Watchtower raised, how it got there and
 * what is at stake; who is on it, or who could take it; the message Triage
 * read; the change an agent asks to make, with its preview; and what deciding
 * it came to. The bar under it acts on it.
 */
export function ItemDetail({
  item,
  group,
  now,
  timezone,
  actions,
  decision,
  canAct,
  canDecide,
  declining,
  onDecliningChange,
  onBack,
}: {
  item: DeskAttentionItem;
  group: DeskAttentionGroup;
  now: number;
  timezone: string;
  actions: WatchtowerActions;
  decision: ItemDecision;
  canAct: boolean;
  canDecide: boolean;
  /** Whether the dialog asking why it is declined is open; the D key opens it too. */
  declining: boolean;
  onDecliningChange: (open: boolean) => void;
  onBack: () => void;
}) {
  const t = useT();
  const navigate = useNavigate();
  const minutes = minutesLeft(item.dueAt, now);
  const countdown = formatCountdown(t, minutes);
  const record = group.record;
  const readable = (record?.readable ?? false) && record?.label !== "";
  const isDecision = item.deskLens === "Decisions";
  // Worked in its case: it is closed there, not cleared or handed off here.
  const caseWork = caseWorkOf(group);
  const isMessage = item.deskLens === "Triage";
  // The case is someone else's, whether they took it or the record gave it to
  // them, and the server refuses anything done to it from here. Nothing is
  // offered that it would refuse, and the bar says whose it is.
  const ownedElsewhere = group.ownership !== null && !group.ownership.canWork;
  const panelSaysIt = item.panel?.kind === "Ask" || item.panel?.kind === "Finding";
  const work = item.agentWork;
  const proposal = decision.proposals[0] ?? null;
  const unseenPreviews = decision.tally.loading + decision.tally.failed + decision.tally.stale;
  const blocked = useRefusal(decision);
  const agentName = work?.agent?.name ?? null;
  const raised = formatFactTime(t, item.occurredAt, now, timezone);
  return (
    <div className="dk-w-det-w">
      <div className="dk-w-det">
        <button type="button" className="dk-w-det-back ui-focus-ring" onClick={onBack}>
          <DeskIcon name="chevL" size={12} />
          {t("All that needs you")}
        </button>
        <div className="dk-w-det-k">
          <span>{item.kindLabel}</span>
          {record && readable && (
            <>
              <span>·</span>
              <span className="dk-w-mono">{record.label}</span>
            </>
          )}
          {recordContext(t, group).map((part) => (
            <span key={part}>· {part}</span>
          ))}
          <span className="dk-w-sp" />
          <LifeBadge life={decision.phase === "done" ? "done" : itemLife(item)} />
        </div>
        <h1 className="dk-w-det-h">{item.title}</h1>
        {/* An agent's ask and an insight say their sentence in their own panel. */}
        {item.summary !== "" && !isMessage && !panelSaysIt && (
          <p className="dk-w-det-sum">{item.summary}</p>
        )}

        {/* A deadline the record itself sets (free time, a window, a start) is the
            clock a person works against, and is drawn large. One set by policy is
            how long the item has waited, which its row already says, so it is one
            quiet line here. */}
        {countdown !== null &&
          // A decision waits on the reader, which its badge already says.
          !(isDecision && item.dueSource !== "Source") &&
          (item.dueSource === "Source" ? (
            <div className="dk-w-clock">
              <span className="dk-w-clock-l">
                {item.dueLabel ||
                  (minutes !== null && minutes < 0 ? t("Past its deadline") : t("Due"))}
              </span>
              <span className={cn("dk-w-clock-v", dueTone(minutes))}>{countdown}</span>
            </div>
          ) : (
            <p className="mt-2 text-dsk-sm text-dsk-subtle">
              {minutes !== null && minutes < 0
                ? t("Waiting on someone, {0} past its target.", formatMinutesSpan(-minutes))
                : t("Due in {0}.", countdown)}
            </p>
          ))}

        {!isDecision && (
          <ItemPanel
            item={item}
            now={now}
            timezone={timezone}
            lockedBy={
              ownedElsewhere
                ? t(
                    "{0} owns this case, so only they can file its papers. A case lead can give it to you.",
                    group.ownership?.owner?.name ?? t("Someone else"),
                  )
                : ""
            }
            readable={record === null || readable}
          />
        )}

        {work && <AgentWorkBlock work={work} />}

        {!work && item.suggestedAgent && !isDecision && (
          <section className="dk-w-blk">
            <div className="dk-w-banner">
              <AssistMark className="size-3.5 text-dsk-subtle" />
              <span>
                {t(
                  "{0} can take this. It reads the record, works out what to do and asks you before it changes anything.",
                  item.suggestedAgent.name,
                )}
              </span>
            </div>
          </section>
        )}

        {isDecision && (
          <DecisionCard
            decision={decision}
            agentName={work?.agent?.name ?? null}
            raised={raised}
            said={item.summary}
            blocked={blocked}
          />
        )}

        {decision.phase === "done" && (
          <div className="dk-w-outcome dk-ok" role="status">
            <DeskIcon name="check" size={14} />
            <div>
              <b>
                {t(
                  "Done. The server confirmed it at {0}.",
                  formatUnixTime(decision.decidedAt ?? now, { timezone }),
                )}
              </b>
            </div>
          </div>
        )}
        {decision.phase === "declined" && (
          <div className="dk-w-outcome dk-neutral" role="status">
            <DeskIcon name="x" size={14} />
            <div>
              <b>{t("Declined. Nothing ran.")}</b>{" "}
              {t("The agent reads your reason in its conversation.")}
            </div>
          </div>
        )}
        {decision.phase === "failed" && (
          <div className="dk-w-outcome dk-bad" role="alert">
            <DeskIcon name="alert" size={14} />
            <div>
              <b>{t("It didn't go through.")}</b> {decision.failure} {t("Nothing changed.")}
              <div className="dk-w-outcome-acts">
                <Button size="sm" variant="outline" onClick={decision.reset}>
                  {t("Try again")}
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>

      <div className="dk-w-bar">
        <span className="dk-w-bar-note">
          {isDecision
            ? blocked.length > 0
              ? t("It can't run as it stands. Sending it back tells the agent why.")
              : t("You approve exactly what is shown here.")
            : isMessage
              ? t("A message is answered, filed or set aside in Triage, which clears it here.")
              : ownedElsewhere && caseWork === null
                ? t(
                    "{0} owns this case, so only they can act on it. You can read it; a case lead can give it to you.",
                    group.ownership?.owner?.name ?? t("Someone else"),
                  )
                : caseWork !== null
                  ? caseWork.owner !== null && !caseWork.mine
                    ? t(
                        "{0} is working this in its case. It is closed there, by approving the plan or accepting the risk.",
                        caseWork.owner.name,
                      )
                    : t(
                        "This is being worked in its case. Close it there, by approving the plan or accepting the risk.",
                      )
                  : t("Done clears it for everyone; it comes back if the problem returns.")}
        </span>
        <div className="dk-w-bar-acts">
          {isDecision && decision.phase === "idle" && canDecide && proposal !== null ? (
            <>
              <ReviewBlocker tally={decision.tally} plan={decision.planId !== null} />
              <Button
                variant="ghost"
                className="text-danger-foreground"
                onClick={() => onDecliningChange(true)}
              >
                {t("Decline")} <Key k="D" />
              </Button>
              {blocked.length > 0 ? (
                // It would be refused if approved, so the way forward is back
                // to the agent with the reasons, in one step.
                <Button
                  onClick={() =>
                    void decision.decline(askAgentMessage(proposal.toolName, blocked, t))
                  }
                >
                  {agentName ? t("Send back to {0}", agentName) : t("Send back to the agent")}
                </Button>
              ) : (
                <Button disabled={unseenPreviews > 0} onClick={() => void decision.approve()}>
                  {t("Approve")} <Key k="A" inv />
                </Button>
              )}
            </>
          ) : isDecision && decision.phase === "deciding" ? (
            <Button isLoading>{t("Approve")}</Button>
          ) : null}
          {!isDecision && !isMessage && canAct && caseWork === null && !ownedElsewhere && (
            <>
              <Button variant="ghost" onClick={actions.snooze}>
                <DeskIcon name="clock" size={13} />
                {t("Snooze")} <Key k="S" />
              </Button>
              {item.agentWork === null && item.subjectId !== null && (
                <WatchtowerHandOff
                  suggested={item.suggestedAgent}
                  open={actions.menuOpen}
                  onOpenChange={actions.setMenuOpen}
                  onHandOff={actions.handOff}
                  disabled={actions.handingOff}
                />
              )}
              <Button variant={readable ? "outline" : "default"} onClick={actions.done}>
                <DeskIcon name="check" size={13} />
                {t("Done")} <Key k="E" inv={!readable} />
              </Button>
            </>
          )}
          {isMessage && item.path !== "" && (
            <Button
              variant={readable ? "secondary" : "default"}
              onClick={() => void navigate(item.path)}
            >
              <DeskIcon name="mail" size={13} />
              {t("Open in Triage")}
              {!readable && <Key k={<CornerDownLeftIcon aria-hidden />} inv />}
            </Button>
          )}
          {record && readable && (
            <Button
              onClick={() =>
                void navigate(casePath(record.kind, record.id), {
                  state: START_CASE,
                  viewTransition: true,
                })
              }
            >
              {t("Open the case")} <Key k={<CornerDownLeftIcon aria-hidden />} inv />
            </Button>
          )}
        </div>
      </div>

      <DeclineDialog
        open={declining}
        agentName={agentName}
        pending={decision.phase === "deciding"}
        onOpenChange={onDecliningChange}
        onDecline={async (reason) => {
          await decision.decline(reason);
          onDecliningChange(false);
        }}
      />
    </div>
  );
}
