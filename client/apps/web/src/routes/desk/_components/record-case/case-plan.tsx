import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { CaseFooter } from "./case-footer";
import { PreviewRemedies } from "@/components/assistant/proposal-preview/preview-remedies";
import { ValueChange } from "@/components/assistant/proposal-preview/value-change";
import { channelLabel } from "@/components/assistant/proposal-preview/preview-format";
import {
  previewWarningText,
  previewWarningTone,
  visibleWarnings,
} from "@/components/assistant/proposal-preview/preview-warnings";
import { isOpaqueId, presentProposal } from "@/components/assistant/proposal-presenters";
import type { PreviewRecordChange, ProposalPreview } from "@/lib/graphql/agent-preview";
import type { DeskCase, DeskCaseDecision } from "@/lib/graphql/desk-case";
import type { AssistantProposal } from "@/types/assistant";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatCountdownClock, formatUnixTime } from "@trenova/shared/lib/date";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import { buttonVariants } from "@trenova/shared/lib/variants/button";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useState, type ReactNode } from "react";
import { Link } from "react-router";
import { isBareId, readableField } from "./plan-values";
import type { CaseDecision } from "./use-case-decision";

type StepState = "ran" | "run" | "bad" | "skipped" | "";

type PlanProps = {
  data: DeskCase;
  decision: DeskCaseDecision;
  state: CaseDecision;
  canDecide: boolean;
  timezone: string;
  recordPath: string | null;
  onDecline: () => void;
  onChange: () => void;
  onAsk: (question: string) => void;
};

/** Where a step stands: the server's own word for it first, then where the plan got to. */
function stepStateOf(
  decision: DeskCaseDecision,
  step: AssistantProposal,
  index: number,
): StepState {
  if (step.status === "Executed") {
    return "ran";
  }
  if (step.status === "ExecutionFailed") {
    return "bad";
  }
  if (step.status === "Skipped") {
    return "skipped";
  }
  switch (decision.phase) {
    case "Done":
      return "ran";
    case "Running":
      return index < decision.completedSteps
        ? "ran"
        : index === decision.completedSteps
          ? "run"
          : "";
    case "Failed": {
      const failed = failedIndex(decision);
      return index < failed ? "ran" : index === failed ? "bad" : "skipped";
    }
    default:
      return "";
  }
}

/**
 * The place in the list of the step that did not go through. A plan names it
 * by number, counted from one; a plan that recorded none stopped at the step
 * after the last that ran.
 */
export function failedIndex(decision: Pick<DeskCaseDecision, "failedStep" | "completedSteps">) {
  return decision.failedStep != null && decision.failedStep > 0
    ? decision.failedStep - 1
    : decision.completedSteps;
}

/** The warnings drawn under a step: every one a plan shows, but the note the plan's foot carries once. */
function stepWarnings(preview: ProposalPreview | null | undefined) {
  return visibleWarnings(preview?.warnings ?? [], { inPlan: true }).filter(
    (warning) => warning.code !== "unpinned",
  );
}

/** Counts down to the moment an approval in its undo window starts to run. */
function useSecondsUntil(at: number | null): number {
  const [now, setNow] = useState(() => Math.floor(Date.now() / 1000));
  useEffect(() => {
    if (at === null) {
      return;
    }
    const timer = setInterval(() => setNow(Math.floor(Date.now() / 1000)), 500);
    return () => clearInterval(timer);
  }, [at]);

  return at === null ? 0 : at - now;
}

function figureLine(t: TranslateFn, previews: ReadonlyMap<string, ProposalPreview>): string {
  const records = new Set<string>();
  let messages = 0;
  for (const preview of previews.values()) {
    if (preview.coverage === "Unavailable") {
      continue;
    }
    for (const change of preview.changes) {
      if (change.message) {
        messages += 1;
      } else {
        records.add(`${change.resource}:${change.entityId}`);
      }
    }
  }
  if (records.size === 0 && messages === 0) {
    return t("changes");
  }
  const parts: string[] = [];
  if (records.size > 0) {
    parts.push(t("{0, plural, one {# record} other {# records}}", records.size));
  }
  if (messages > 0) {
    parts.push(t("{0, plural, one {# message} other {# messages}}", messages));
  }

  return t("changes · {0}", parts.join(", "));
}

/**
 * The plan to decide: what it is, how many changes, where it stands, each
 * step with the record it changes and what was there before, and the way to
 * approve or decline exactly that. An approval waits out its undo window
 * before anything runs, and nothing is marked run until the server says so.
 */
export function CasePlan({
  data,
  decision,
  state,
  canDecide,
  timezone,
  recordPath,
  onDecline,
  onChange,
  onAsk,
}: PlanProps) {
  const t = useT();
  const steps = state.steps;
  const count = Math.max(decision.stepCount, steps.length);
  const agentName = decision.agent?.name ?? t("The agent");
  const permanent = steps.some((step) => !presentProposal(step).reversible);
  const secondsLeft = useSecondsUntil(decision.phase === "Undoable" ? decision.commitsAt : null);
  const failedAt = failedIndex(decision);
  // One change has no title of its own: it is named by what it does, in the
  // words the step itself is drawn with.
  const only = decision.kind === "Proposal" ? steps[0] : undefined;
  const heading = only ? t(presentProposal(only).title) : decision.title;
  const warned = [...state.previews.values()].some((preview) => stepWarnings(preview).length > 0);
  // Said once for the plan, at its foot, rather than under every step.
  const unpinned = [...state.previews.values()].some((preview) =>
    preview.warnings.some((warning) => warning.code === "unpinned"),
  );

  const status =
    decision.phase === "Awaiting" ? (
      <Badge variant="warning" dot>
        {t("Awaiting your approval")}
      </Badge>
    ) : decision.phase === "Undoable" ? (
      <Badge variant="info" dot>
        {t("Approved · nothing has run yet")}
      </Badge>
    ) : decision.phase === "Running" ? (
      <Badge variant="info">
        <span className="dk-cz-spin" />
        {t("Running step {0} of {1}", Math.min(decision.completedSteps + 1, count), count)}
      </Badge>
    ) : decision.phase === "Done" ? (
      <Badge variant="success">
        <DeskIcon name="check" size={11} />
        {decision.decidedAt
          ? t("Done · confirmed {0}", formatUnixTime(decision.decidedAt, { timezone }))
          : t("Done")}
      </Badge>
    ) : decision.phase === "Failed" ? (
      <Badge variant="danger">
        <DeskIcon name="alert" size={11} />
        {t("Stopped at step {0}", failedAt + 1)}
      </Badge>
    ) : (
      <Badge dot>
        {decision.decidedBy ? t("Declined by {0}", decision.decidedBy.name) : t("Declined")}
      </Badge>
    );

  const checked =
    decision.phase !== "Awaiting"
      ? null
      : state.previewLoading
        ? t("Checking each step")
        : state.stale
          ? t("Checked: a record changed since this was drafted")
          : state.previewFailed
            ? t("The steps could not be checked")
            : state.refusedStep !== null
              ? count > 1
                ? t("Checked: step {0} would be refused as things stand", state.refusedStep + 1)
                : t("Checked: it would be refused as things stand")
              : warned
                ? t("Checked: would go through, with warnings")
                : t("Checked: would go through as shown");

  return (
    <section className="dk-cz-plan" aria-label={t("Plan")} tabIndex={-1} id="dk-cz-plan">
      <div className="dk-cz-plan-h">
        <div className="dk-cz-plan-k">
          {count > 1
            ? t("Plan · {0} steps · runs in order, stops at the first failure", count)
            : t("One change")}
        </div>
        <div className="dk-cz-plan-row">
          <h3>{heading}</h3>
          <div className="dk-cz-plan-fig">
            <b>{count}</b>
            <small>{figureLine(t, state.previews)}</small>
          </div>
        </div>
        <div
          className="dk-cz-segs"
          style={{ gridTemplateColumns: `repeat(${Math.max(count, 1)}, 1fr)` }}
        >
          {Array.from({ length: count }, (_, index) => {
            const step = steps[index];
            const stepState = step ? stepStateOf(decision, step, index) : "";
            return (
              <i
                key={index}
                className={cn(stepState !== "" && stepState !== "skipped" && `dk-${stepState}`)}
              />
            );
          })}
        </div>
        <div className="dk-cz-plan-meta">
          {status}
          <span>{t("Proposed by {0}", agentName)}</span>
          <span aria-hidden>·</span>
          <span>{formatUnixTime(decision.proposedAt, { timezone })}</span>
          {permanent && (
            <Badge variant="warning" dot>
              {t("Permanent")}
            </Badge>
          )}
          {checked && <span aria-hidden>·</span>}
          {checked && <span>{checked}</span>}
        </div>
      </div>

      {state.stepsLoading ? (
        <div className="dk-cz-draft" aria-busy aria-label={t("Reading the plan")}>
          <i className="dk-sk" />
          <i className="dk-sk" />
        </div>
      ) : state.stepsFailed ? (
        <div className="dk-cz-draft">
          <span className="dk-cz-draft-l">{t("The plan's steps could not be read.")}</span>
        </div>
      ) : (
        steps.map((step, index) => (
          <PlanStep
            key={step.id}
            step={step}
            index={index}
            decision={decision}
            preview={state.previews.get(step.id) ?? null}
            editable={decision.phase === "Awaiting" && canDecide}
            onChange={onChange}
          />
        ))
      )}

      {decision.phase === "Awaiting" && canDecide && (
        <CaseFooter
          lead={
            state.failure !== "" && (
              <div className="dk-w-outcome dk-bad mb-2">
                <DeskIcon name="alert" size={14} />
                <div>
                  <b>{t("That didn't go through.")}</b> {state.failure}
                </div>
              </div>
            )
          }
          note={
            <>
              {t("You approve exactly what is shown here.")}{" "}
              {permanent && t("The steps marked can't be taken back once they run.")}{" "}
              {state.stale &&
                t("A record changed since this was drafted, so it can only be declined.")}{" "}
              {unpinned &&
                t(
                  "It was proposed without noting each record's version, so an edit made since can't be detected.",
                )}{" "}
              {!state.stale &&
                state.refusedStep !== null &&
                (count > 1
                  ? t(
                      "As things stand step {0} would be refused, and the plan would stop there.",
                      state.refusedStep + 1,
                    )
                  : t("As things stand it would be refused."))}
            </>
          }
        >
          {!state.stale && state.refusedStep !== null && (
            <Button
              variant="outline"
              disabled={state.busy !== null}
              onClick={() =>
                onAsk(
                  t(
                    "Step {0} of this plan would be refused as things stand. Redraft it so every step can run.",
                    state.refusedStep === null ? 1 : state.refusedStep + 1,
                  ),
                )
              }
            >
              {t("Ask {0} to redraft", agentName)}
            </Button>
          )}
          <Button
            variant="ghost"
            className="text-danger-foreground"
            disabled={state.busy !== null}
            isLoading={state.busy === "decline"}
            shortcut="D"
            onClick={onDecline}
          >
            {t("Decline")}
          </Button>
          <Button variant="outline" disabled={state.busy !== null} onClick={onChange}>
            {t("Change values")}
          </Button>
          <Button
            disabled={state.blocked || state.busy !== null || steps.length === 0}
            isLoading={state.busy === "approve"}
            shortcut={formatShortcut("↵")}
            onClick={() => void state.approve()}
          >
            {count > 1 ? t("Approve plan") : t("Approve")}
          </Button>
        </CaseFooter>
      )}
      {decision.phase === "Awaiting" && !canDecide && (
        <CaseFooter note={t("This waits on someone who may approve what agents propose.")} />
      )}
      {decision.phase === "Undoable" && (
        <CaseFooter
          plain
          note={
            <div className="dk-w-outcome dk-blue mt-0!">
              <DeskIcon name="clock" size={14} />
              <div>
                <b>{t("Approved. Nothing has run yet.")}</b>{" "}
                {secondsLeft > 0
                  ? t("It starts in {0}.", formatCountdownClock(secondsLeft))
                  : t("It is starting.")}
              </div>
            </div>
          }
        >
          <Button
            variant="outline"
            autoFocus
            disabled={state.busy !== null}
            isLoading={state.busy === "undo"}
            onClick={() => void state.undo()}
          >
            <DeskIcon name="undo" size={13} />
            {t("Undo")}
          </Button>
          <Button
            variant="ghost"
            disabled={state.busy !== null}
            isLoading={state.busy === "commit"}
            onClick={() => void state.doNow()}
          >
            {t("Do it now")}
          </Button>
        </CaseFooter>
      )}
      {decision.phase === "Running" && (
        <CaseFooter
          note={t(
            "Each step is marked Ran only once the server confirms it. Nothing here is assumed.",
          )}
        />
      )}
      {decision.phase === "Done" && (
        <div className="dk-cz-plan-f dk-block">
          <div className="dk-w-outcome dk-ok">
            <DeskIcon name="check" size={14} />
            <div>
              <b>
                {t(
                  "{0, plural, one {It ran and was confirmed.} other {All # steps ran and were confirmed.}}",
                  count,
                )}
              </b>{" "}
              {data.open > 0
                ? t(
                    "What this plan answered is closed. Anything still open about the record is in Needs you.",
                  )
                : t("Nothing else is open about this record.")}
            </div>
          </div>
        </div>
      )}
      {decision.phase === "Failed" && (
        <div className="dk-cz-plan-f dk-block">
          <div className="dk-w-outcome dk-bad">
            <DeskIcon name="alert" size={14} />
            <div>
              <b>
                {count > 1
                  ? t("Step {0} didn't go through, so the plan stopped.", failedAt + 1)
                  : t("It didn't go through.")}
              </b>{" "}
              {failedAt > 0 &&
                t(
                  "{0, plural, one {Step 1 ran.} other {Steps 1 to # ran.}} The steps after it were not run.",
                  failedAt,
                )}
              <div className="dk-w-outcome-acts">
                <Button
                  size="sm"
                  onClick={() =>
                    onAsk(
                      t(
                        "The plan stopped at step {0}: {1} Redraft it against the record as it is now.",
                        failedAt + 1,
                        decision.failureError,
                      ),
                    )
                  }
                >
                  {t("Ask {0} to redraft", agentName)}
                </Button>
                {recordPath && (
                  <Link
                    className={buttonVariants({ variant: "outline", size: "sm" })}
                    to={recordPath}
                  >
                    {t("Finish it by hand")}
                  </Link>
                )}
                {failedAt > 0 && (
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() =>
                      onAsk(
                        t(
                          "The plan stopped after {0, plural, one {step 1} other {# steps}} ran. Draft the changes that put back what ran.",
                          failedAt,
                        ),
                      )
                    }
                  >
                    {t("{0, plural, one {Undo step 1} other {Undo steps 1 to #}}", failedAt)}
                  </Button>
                )}
              </div>
            </div>
          </div>
        </div>
      )}
      {decision.phase === "Declined" && (
        <div className="dk-cz-plan-f dk-block">
          <div className="dk-w-outcome dk-neutral">
            <DeskIcon name="x" size={14} />
            <div>
              <b>{t("Declined. Nothing ran.")}</b>{" "}
              {decision.note !== "" && t("{0} read the reason: “{1}”.", agentName, decision.note)}{" "}
              {t(
                "The risk stays in Needs you until it clears or you mark it done; every change is still yours to make on the record.",
              )}
              <div className="dk-w-outcome-acts">
                {recordPath && (
                  <Link
                    className={buttonVariants({ variant: "outline", size: "sm" })}
                    to={recordPath}
                  >
                    {t("Open the record")}
                  </Link>
                )}
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() =>
                    onAsk(
                      t(
                        "I declined the plan but want to look at it again. Draft it once more against the record as it is now.",
                      ),
                    )
                  }
                >
                  {t("Reconsider")}
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </section>
  );
}

function PlanStep({
  step,
  index,
  decision,
  preview,
  editable,
  onChange,
}: {
  step: AssistantProposal;
  index: number;
  decision: DeskCaseDecision;
  preview: ProposalPreview | null;
  editable: boolean;
  onChange: () => void;
}) {
  const t = useT();
  const view = presentProposal(step);
  const state = stepStateOf(decision, step, index);
  const label =
    state === "ran"
      ? t("It ran")
      : state === "run"
        ? t("Running…")
        : state === "bad"
          ? t("Didn't go through")
          : state === "skipped"
            ? t("Not run")
            : decision.phase === "Undoable"
              ? t("Approved")
              : decision.phase === "Declined"
                ? t("Won't run")
                : t("Waiting");

  return (
    <div className="dk-cz-step">
      <span className={cn("dk-cz-step-nd", state !== "" && state !== "skipped" && `dk-${state}`)}>
        {state === "ran" ? <DeskIcon name="check" size={12} /> : index + 1}
      </span>
      <span className="dk-cz-step-t">
        {t(view.title)}
        {!view.reversible && (
          <Badge variant="warning" dot className="ml-1.5 align-[1px]">
            {t("Can't be undone")}
          </Badge>
        )}
      </span>
      <span
        className={cn(
          "dk-cz-step-st",
          state === "ran" && "dk-ok",
          state === "run" && "dk-run",
          state === "bad" && "dk-bad",
        )}
      >
        {label}
      </span>
      {preview && preview.coverage !== "Unavailable" && preview.changes.length > 0 ? (
        preview.changes.map((change, changeIndex) => (
          <StepChange
            key={`${change.resource}:${change.entityId}:${changeIndex}`}
            change={change}
            editable={editable && view.reversible}
            onChange={onChange}
          />
        ))
      ) : (
        <div className="dk-cz-chg">
          <div className="dk-cz-chg-h">
            <DeskIcon name={view.reversible ? "file" : "mail"} size={12} />
            {view.summary || t(view.title)}
          </div>
          {view.highlights
            .filter((fact) => !isOpaqueId(String(fact.value)))
            .map((fact) => (
              <div className="dk-cz-chg-r" key={fact.label}>
                <span className="dk-cz-chg-k">{fact.label}</span>
                <span className="dk-cz-chg-v dk-cz-nw whitespace-pre-wrap">
                  {String(fact.value)}
                </span>
              </div>
            ))}
        </div>
      )}
      {decision.phase === "Awaiting" &&
        stepWarnings(preview).map((warning, at) => (
          <div
            // oxlint-disable-next-line react/no-array-index-key -- a preview's warnings keep their order
            key={at}
            className={cn("dk-cz-step-warn", previewWarningTone(warning) === "danger" && "dk-bad")}
            role={previewWarningTone(warning) === "danger" ? "alert" : undefined}
          >
            <DeskIcon name="alert" size={12} />
            <span>
              {previewWarningText(warning, t)}
              <PreviewRemedies warning={warning} />
            </span>
          </div>
        ))}
      {state === "bad" && decision.failureError !== "" && (
        <div className="dk-cz-step-why">{decision.failureError}</div>
      )}
    </div>
  );
}

/** One record a step changes: what it is, then each value as it was and as it would be. */
function StepChange({
  change,
  editable,
  onChange,
}: {
  change: PreviewRecordChange;
  editable: boolean;
  onChange: () => void;
}) {
  const t = useT();
  const message = change.message;

  return (
    <div className="dk-cz-chg">
      <div className="dk-cz-chg-h">
        <DeskIcon name={message ? "mail" : "file"} size={12} />
        {message
          ? t("{0} · {1}", channelLabel(message.channel, t), message.to.join(", "))
          : change.label || change.resource}
        {editable && !message && (
          <Button variant="ghost" size="sm" className="ml-auto h-5 px-2" onClick={onChange}>
            {t("Change")}
          </Button>
        )}
      </div>
      {change.withheld ? (
        <div className="dk-cz-chg-r">
          <span className="dk-cz-chg-k">{t("Values")}</span>
          <span className="dk-cz-chg-v">{t("Hidden by your data access")}</span>
        </div>
      ) : (
        <>
          {message && (
            <>
              {message.subject !== "" && (
                <div className="dk-cz-chg-r">
                  <span className="dk-cz-chg-k">{t("Subject")}</span>
                  <span className="dk-cz-chg-v dk-cz-nw">{message.subject}</span>
                </div>
              )}
              <div className="dk-cz-chg-r">
                <span className="dk-cz-chg-k">{t("Message")}</span>
                <span className="dk-cz-chg-v dk-cz-nw whitespace-pre-wrap">
                  {message.body}
                  {message.bodyTruncated && <> {t("(shortened)")}</>}
                </span>
              </div>
            </>
          )}
          {change.fields
            .filter((field) => !isBareId(field))
            .map((field) => (
              <div className="dk-cz-chg-r" key={field.path}>
                <span className="dk-cz-chg-k">{field.label}</span>
                <span className="dk-cz-chg-v">
                  <ValueChange change={readableField(field)} operation={change.operation} />
                </span>
              </div>
            ))}
          {change.omittedFields > 0 && (
            <div className="dk-cz-chg-r">
              <span className="dk-cz-chg-k" />
              <span className="dk-cz-chg-v dk-cz-chg-k">
                {t(
                  "{0, plural, one {and # more value} other {and # more values}}",
                  change.omittedFields,
                )}
              </span>
            </div>
          )}
        </>
      )}
    </div>
  );
}

/** The plan while the agent drafts the option the person chose. */
export function CasePlanDrafting({ agentName }: { agentName: string }) {
  const t = useT();

  return (
    <section className="dk-cz-plan" aria-label={t("Plan")} aria-busy>
      <div className="dk-cz-plan-h">
        <div className="dk-cz-plan-k">{t("Plan")}</div>
      </div>
      <div className="dk-cz-draft">
        <div className="dk-cz-draft-l">
          <NovaBadge size="xs" />
          {t("{0} is drafting this option against your records as they are now…", agentName)}
        </div>
        <i className="dk-sk" />
        <i className="dk-sk" />
      </div>
    </section>
  );
}

/** No plan: the person chose to leave the record as it is, or none was drafted. */
export function CasePlanNone({
  title,
  note,
  action,
}: {
  title: string;
  note: ReactNode;
  action: { label: string; pending: boolean; onClick: () => void } | null;
}) {
  const t = useT();

  return (
    <section className="dk-cz-plan" aria-label={t("Plan")}>
      <div className="dk-cz-plan-h">
        <div className="dk-cz-plan-k">{t("No plan")}</div>
        <div className="dk-cz-plan-row">
          <h3>{title}</h3>
        </div>
      </div>
      <CaseFooter note={note}>
        {action && (
          <Button variant="outline" isLoading={action.pending} onClick={action.onClick}>
            {action.label}
          </Button>
        )}
      </CaseFooter>
    </section>
  );
}
