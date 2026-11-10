import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import {
  ReviewBlocker,
  ReviewedChange,
  ReviewedPlan,
} from "@/components/assistant/proposal-preview/reviewed-change";
import { presentProposal } from "@/components/assistant/proposal-presenters";
import type { DeskAttentionGroup, DeskAttentionItem } from "@/lib/graphql/desk-attention";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { Link, useNavigate } from "react-router";
import { WatchtowerHandOff } from "../watchtower/hand-off-menu";
import { formatCountdown, formatFactTime, minutesLeft } from "../watchtower/lanes";
import type { WatchtowerActions } from "../watchtower/use-watchtower-actions";
import { RunSteps } from "./run-steps";
import { casePath } from "./attention-record";
import { dueTone, recordContext } from "./attention-row";
import { DeclineDialog } from "./decline-dialog";
import { LifeBadge, itemLife } from "./life-badge";
import type { ItemDecision } from "./use-item-decision";

function Block({
  icon,
  title,
  source,
  children,
}: {
  icon?: ReactNode;
  title: ReactNode;
  source?: string;
  children: ReactNode;
}) {
  return (
    <section className="dk-w-blk">
      <div className="dk-w-blk-h">
        {icon}
        {title}
        {source && <span className="dk-w-blk-src">{source}</span>}
      </div>
      {children}
    </section>
  );
}

function Rows({ rows }: { rows: readonly { key: string; label: string; value: ReactNode }[] }) {
  return (
    <div className="dk-w-kvrows">
      {rows.map((row) => (
        <div key={row.key}>
          <span className="dk-w-kvrows-k">{row.label}</span>
          <span>{row.value}</span>
        </div>
      ))}
    </div>
  );
}

/** Kbd-style hint inside a button, inverted on an ink button. */
function Key({ k, inv = false }: { k: string; inv?: boolean }) {
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
  const isMessage = item.deskLens === "Triage";
  const isCondition = item.deskLens === "Watchtower";
  const work = item.agentWork;
  const proposal = decision.proposals[0] ?? null;
  const view = proposal ? presentProposal(proposal) : null;
  const unseenPreviews = decision.tally.loading + decision.tally.failed + decision.tally.stale;
  const raised = formatFactTime(t, item.occurredAt, now, timezone);
  const conditionRows = [
    ...item.impact.map((stake, index) => ({
      key: `i${index}`,
      label: t("At stake"),
      value: stake,
    })),
    ...item.facts.map((fact, index) => ({
      key: `f${index}`,
      label: fact.label,
      value: fact.value ?? (fact.at !== null ? formatFactTime(t, fact.at, now, timezone) : ""),
    })),
  ];

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
        {item.summary !== "" && !isMessage && <p className="dk-w-det-sum">{item.summary}</p>}

        {countdown !== null && (
          <div className="dk-w-clock">
            <span className="dk-w-clock-l">
              {item.dueLabel ||
                (minutes !== null && minutes < 0 ? t("Past its deadline") : t("Due"))}
            </span>
            <span className={cn("dk-w-clock-v", dueTone(minutes))}>{countdown}</span>
          </div>
        )}

        {isCondition && (conditionRows.length > 0 || item.trail.length > 0) && (
          <Block
            icon={<DeskIcon name="radar" size={14} />}
            title={t("Condition")}
            source={t("Watchtower · raised {0}", raised)}
          >
            {conditionRows.length > 0 && <Rows rows={conditionRows} />}
            {item.trail.length > 0 && (
              <ol className="dk-w-trail" aria-label={t("How it got here")}>
                {item.trail.map((step, index) => (
                  <li key={index}>
                    <span className="dk-w-mono">{formatUnixTime(step.at, { timezone })}</span>
                    {step.text}
                  </li>
                ))}
              </ol>
            )}
          </Block>
        )}

        {work && (
          <Block
            icon={<DeskAgentTile agent={work.agent} size="xs" />}
            title={
              work.failed
                ? t("{0} could not finish", work.agent?.name ?? t("The agent"))
                : t("{0} is on it", work.agent?.name ?? t("An agent"))
            }
          >
            <div className="dk-w-run dk-w-run-flat">
              <DeskAgentTile agent={work.agent} size="sm" />
              <span className="dk-w-run-main">
                <span className="dk-w-run-ti">{item.title}</span>
                <span className="dk-w-run-nt">
                  {work.note !== "" ? work.note : t("It will ask you before it changes anything.")}
                </span>
              </span>
              <RunSteps work={work} />
            </div>
          </Block>
        )}

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

        {isMessage && (
          <Block
            icon={<DeskIcon name="mail" size={14} />}
            title={t("Message")}
            source={t("Triage · {0}", raised)}
          >
            <div className="dk-w-msg">
              <div className="dk-w-msg-h">
                <b>{item.facts.find((fact) => fact.value)?.value ?? item.kindLabel}</b>
                <span className="dk-w-mono">{formatUnixTime(item.occurredAt, { timezone })}</span>
              </div>
              <div className="dk-w-msg-b">{item.summary || item.title}</div>
              <div className="dk-w-msg-f">
                <NovaBadge size="xs" />
                <span>{t("Nova read it as: {0}.", item.kindLabel)}</span>
                {item.path !== "" && (
                  <Link className="dk-w-alnk ui-focus-ring" to={item.path}>
                    {t("Open in Triage")}
                  </Link>
                )}
              </div>
            </div>
          </Block>
        )}

        {isDecision && (
          <Block
            icon={<DeskIcon name="inbox" size={14} />}
            title={t("Decision")}
            source={
              work?.agent
                ? t("Proposed by {0} · {1}", work.agent.name, raised)
                : t("Proposed {0}", raised)
            }
          >
            {decision.loading ? (
              <div className="dk-w-sk" aria-busy aria-label={t("Reading the change")}>
                <i className="dk-sk" />
              </div>
            ) : decision.failedToRead || proposal === null ? (
              <p className="dk-w-quiet">
                {t("The change could not be read. Open it on Decisions.")}
              </p>
            ) : (
              <>
                <Rows
                  rows={[
                    {
                      key: "change",
                      label: t("Would change"),
                      value:
                        decision.proposals.length > 1
                          ? t(
                              "{0, plural, one {# change} other {# changes}}, in order: {1}",
                              decision.proposals.length,
                              decision.proposals
                                .map((step) => presentProposal(step).title)
                                .join(" · "),
                            )
                          : (view?.summary ?? view?.title ?? ""),
                    },
                    ...(proposal.rationale
                      ? [{ key: "why", label: t("Why it proposes it"), value: proposal.rationale }]
                      : []),
                    {
                      key: "undo",
                      label: t("Can it be undone"),
                      value: decision.proposals.every((step) => presentProposal(step).reversible)
                        ? t("Yes, it can be reversed after it runs.")
                        : t("No. Once it runs it stays done."),
                    },
                    {
                      key: "asks",
                      label: t("Why it asks"),
                      value: t("AI control needs a person to approve this change before it runs."),
                    },
                  ]}
                />
                {decision.phase === "idle" && decision.reviewed && (
                  <div className="dk-w-preview">
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
                      />
                    )}
                  </div>
                )}
              </>
            )}
          </Block>
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
            ? t("You approve exactly what is shown here.")
            : t("Done clears it for everyone; it comes back if the problem returns.")}
        </span>
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
            <Button disabled={unseenPreviews > 0} onClick={() => void decision.approve()}>
              {t("Approve")} <Key k="A" inv />
            </Button>
          </>
        ) : isDecision && decision.phase === "deciding" ? (
          <Button isLoading>{t("Approve")}</Button>
        ) : null}
        {!isDecision && canAct && (
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
        {record && readable && (
          <Button onClick={() => void navigate(casePath(record.kind, record.id))}>
            {t("Open the case")} <Key k="↵" inv />
          </Button>
        )}
      </div>

      <DeclineDialog
        open={declining}
        agentName={work?.agent?.name ?? null}
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
