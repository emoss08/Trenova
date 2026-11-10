import { DeskIcon } from "@/components/desk-chat/desk-icons";
import {
  ReviewBlocker,
  ReviewedChange,
  ReviewedPlan,
} from "@/components/assistant/proposal-preview/reviewed-change";
import { presentProposal } from "@/components/assistant/proposal-presenters";
import type { DeskAttentionItem } from "@/lib/graphql/desk-attention";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import { cn } from "@trenova/shared/lib/utils";
import type { ItemDecision } from "../attention/use-item-decision";

/**
 * The change an agent asks to make about the record, in the plan card's
 * anatomy and in plain words: what it is, how many changes, where it stands,
 * each step with what it sets, and the way to approve or decline exactly
 * that. Nothing is marked run until the server answers.
 */
export function CasePlan({
  item,
  decision,
  canDecide,
  timezone,
  onDecline,
}: {
  item: DeskAttentionItem;
  decision: ItemDecision;
  canDecide: boolean;
  timezone: string;
  onDecline: () => void;
}) {
  const t = useT();
  const steps = decision.proposals;
  const unseen = decision.tally.loading + decision.tally.failed + decision.tally.stale;
  const agentName = item.agentWork?.agent?.name ?? null;
  const stepState = (index: number) =>
    decision.phase === "done"
      ? "ran"
      : decision.phase === "deciding"
        ? index === 0
          ? "run"
          : ""
        : "";

  const status =
    decision.phase === "done" ? (
      <span className="dk-w-bdg dk-ok">
        <DeskIcon name="check" size={11} />
        {t("Done · confirmed {0}", formatUnixTime(decision.decidedAt ?? 0, { timezone }))}
      </span>
    ) : decision.phase === "failed" ? (
      <span className="dk-w-bdg dk-crit">
        <DeskIcon name="alert" size={11} />
        {t("Didn't go through")}
      </span>
    ) : decision.phase === "declined" ? (
      <span className="dk-w-bdg">{t("Declined by you")}</span>
    ) : decision.phase === "deciding" ? (
      <span className="dk-w-bdg dk-blue">
        <span className="dk-w-spin" />
        {t("Running")}
      </span>
    ) : (
      <span className="dk-w-bdg dk-warn">
        <span className="dk-w-dot dk-warn" />
        {t("Awaiting your approval")}
      </span>
    );

  return (
    <section className="dk-w-plan" aria-label={t("Plan")} id="dk-w-plan">
      <div className="dk-w-plan-h">
        <div className="dk-w-plan-k">
          {steps.length > 1
            ? t("Plan · {0} steps · runs in order, stops at the first failure", steps.length)
            : t("Change")}
        </div>
        <div className="dk-w-plan-row">
          <h3>{item.title}</h3>
          <div className="dk-w-plan-fig">
            <span className="dk-w-mono">{steps.length}</span>
            <small>{t("{0, plural, one {change} other {changes}}", steps.length)}</small>
          </div>
        </div>
        <div
          className="dk-w-segs"
          style={{ gridTemplateColumns: `repeat(${Math.max(steps.length, 1)}, 1fr)` }}
        >
          {steps.map((step, index) => (
            <i key={step.id} className={cn(stepState(index) && `dk-${stepState(index)}`)} />
          ))}
        </div>
        <div className="dk-w-plan-meta">
          {status}
          {agentName && <span>{t("Proposed by {0}", agentName)}</span>}
          <span>·</span>
          <span>{formatUnixTime(item.occurredAt, { timezone })}</span>
        </div>
      </div>

      {decision.loading ? (
        <div className="dk-w-sk dk-w-plan-sk" aria-busy aria-label={t("Reading the plan")}>
          <i className="dk-sk" />
          <i className="dk-sk" />
        </div>
      ) : decision.failedToRead ? (
        <p className="dk-w-quiet dk-w-plan-sk">{t("The plan could not be read.")}</p>
      ) : (
        steps.map((step, index) => {
          const view = presentProposal(step);
          const state = stepState(index);
          return (
            <div className="dk-w-step" key={step.id}>
              <span className={cn("dk-w-step-nd", state && `dk-${state}`)}>
                {state === "ran" ? <DeskIcon name="check" size={12} /> : index + 1}
              </span>
              <span className="dk-w-step-t">
                {t(view.title)}
                {!view.reversible && (
                  <span className="dk-w-bdg dk-outl dk-w-step-perm">{t("Can't be undone")}</span>
                )}
              </span>
              <span
                className={cn(
                  "dk-w-step-st",
                  state === "ran" && "dk-ok",
                  state === "run" && "dk-run",
                )}
              >
                {state === "ran"
                  ? t("Went through")
                  : state === "run"
                    ? t("Running…")
                    : t("Waiting")}
              </span>
              {view.highlights.length > 0 && (
                <div className="dk-w-chg">
                  <div className="dk-w-chg-h">
                    <DeskIcon name="file" size={12} />
                    {view.summary}
                  </div>
                  {view.highlights.map((fact) => (
                    <div className="dk-w-chg-r" key={fact.label}>
                      <span className="dk-w-chg-k">{fact.label}</span>
                      <span className="dk-w-chg-nw">{String(fact.value)}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          );
        })
      )}

      {decision.phase === "idle" && decision.reviewed && steps[0] && (
        <div className="dk-w-plan-prev">
          {decision.planId ? (
            <ReviewedPlan planId={decision.planId} onReport={decision.onReport} scope="approver" />
          ) : (
            <ReviewedChange proposal={steps[0]} onReport={decision.onReport} scope="approver" />
          )}
        </div>
      )}

      {decision.phase === "idle" && canDecide && steps.length > 0 && (
        <div className="dk-w-plan-f">
          <span className="dk-w-plan-note">{t("You approve exactly what is shown here.")}</span>
          <ReviewBlocker tally={decision.tally} plan={decision.planId !== null} />
          <Button variant="ghost" className="text-danger-foreground" onClick={onDecline}>
            {t("Decline")} <span className="dk-w-kbd">D</span>
          </Button>
          <Button disabled={unseen > 0} onClick={() => void decision.approve()}>
            {steps.length > 1 ? t("Approve plan") : t("Approve")}
            <span className="dk-w-kbd dk-inv">{formatShortcut("↵")}</span>
          </Button>
        </div>
      )}
      {decision.phase === "deciding" && (
        <div className="dk-w-plan-f">
          <span className="dk-w-plan-note">
            {t(
              "Each step is marked Ran only once the server confirms it. Nothing here is assumed.",
            )}
          </span>
        </div>
      )}
      {decision.phase === "done" && (
        <div className="dk-w-plan-f dk-w-plan-f-block">
          <div className="dk-w-outcome dk-ok">
            <DeskIcon name="check" size={14} />
            <div>
              <b>
                {t(
                  "{0, plural, one {It ran and was confirmed.} other {All # steps ran and were confirmed.}}",
                  steps.length,
                )}
              </b>{" "}
              {t("Watchtower clears the condition on its own once the record is back on track.")}
            </div>
          </div>
        </div>
      )}
      {decision.phase === "failed" && (
        <div className="dk-w-plan-f dk-w-plan-f-block">
          <div className="dk-w-outcome dk-bad">
            <DeskIcon name="alert" size={14} />
            <div>
              <b>{t("It didn't go through, so the plan stopped.")}</b> {decision.failure}
              <div className="dk-w-outcome-acts">
                <Button size="sm" variant="outline" onClick={decision.reset}>
                  {t("Try again")}
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
      {decision.phase === "declined" && (
        <div className="dk-w-plan-f dk-w-plan-f-block">
          <div className="dk-w-outcome dk-neutral">
            <DeskIcon name="x" size={14} />
            <div>
              <b>{t("Declined. Nothing ran.")}</b>{" "}
              {t(
                "The risk stays in Needs you until it clears or you mark it done; every change is still yours to make on the record.",
              )}
            </div>
          </div>
        </div>
      )}
    </section>
  );
}
