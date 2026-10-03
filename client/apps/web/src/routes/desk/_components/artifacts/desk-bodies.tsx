import { decisionRequestOf } from "@/components/assistant/decision-requests";
import { RequestedDecisionRecords } from "@/components/assistant/decision-record";
import { DisplayValue } from "@/components/assistant/display-value";
import {
  formatDisplayValue,
  isDetailType,
  statusPhase,
} from "@/components/assistant/readable-values";
import { ReportRunCard } from "@/components/assistant/report-run-card";
import { AiMarkdown } from "@/components/elements/ai-markdown";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { queries } from "@/lib/queries";
import type { AssistantArtifact, AssistantProposal } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { humanizeToolName } from "@/components/assistant/proposal-state";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import {
  composedViewFrom,
  documentFrom,
  emailDraftFrom,
  entityCardFrom,
  navigationFrom,
  planFrom,
  rateExplanationFrom,
  reportRunFrom,
  runDiffFrom,
} from "./artifact-payloads";
import { ArtIcon } from "./desk-art-kinds";
import { DeskRecordView } from "./desk-record-view";

const PHASE_PILL: Record<string, string> = {
  failed: "dk-warn",
  attention: "dk-warn",
  awaiting: "dk-blue",
  active: "dk-blue",
  queued: "dk-blue",
  complete: "dk-green",
  closed: "dk-ink",
};

function money(amount: string, currency: string): string {
  const figure = Number(amount);
  if (amount === "" || !Number.isFinite(figure)) {
    return amount;
  }
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: currency || "USD",
    minimumFractionDigits: 2,
  }).format(figure);
}

/** A notice in place of a body, for an artifact that holds nothing to show. */
function Notice({ children }: { children: React.ReactNode }) {
  return <div className="dk-ax-pad dk-ax-oldnote">{children}</div>;
}

/** A record as its name, its status and its fields, with a way into the record itself. */
export function DeskRecordBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const card = useMemo(() => entityCardFrom(artifact), [artifact]);
  const [lead, ...rest] = card.fields;
  const status = rest.find((field) => field.type === "status");
  const fields = rest.filter((field) => field !== status && !isDetailType(field.type));
  const details = rest.filter((field) => isDetailType(field.type));
  const title = lead ? formatDisplayValue(lead.type, lead.value, t) : artifact.title;
  const statusText = status ? formatDisplayValue(status.type, status.value, t) : "";
  const phase = status && typeof status.value === "string" ? statusPhase(status.value) : null;

  if (card.view) {
    return <DeskRecordView title={title} view={card.view} entity={card.entity} path={card.path} />;
  }

  return (
    <div className="dk-ax-pad dk-ax-rec">
      <div className="dk-ax-rec-h">
        <div>
          <div className="dk-ax-big">{title}</div>
          {card.entity !== "" && <div className="dk-ax-sub">{humanizeToolName(card.entity)}</div>}
        </div>
        {statusText !== "" && (
          <span className={cn("dk-ax-pill dk-lg", phase ? PHASE_PILL[phase] : "")}>
            <i />
            {statusText}
          </span>
        )}
      </div>
      <dl className="dk-ax-fields">
        {fields.map((field) => (
          <div key={field.key}>
            <dt>{field.label}</dt>
            <dd>
              <DisplayValue type={field.type} value={field.value} label={field.label} inline />
            </dd>
          </div>
        ))}
      </dl>
      {details.map((field) => (
        <div key={field.key} className="dk-ax-note">
          <b>{field.label}</b>{" "}
          <DisplayValue type={field.type} value={field.value} label={field.label} />
        </div>
      ))}
      {card.path !== "" && (
        <Link className="dk-ax-btn" to={card.path}>
          <ArtIcon name="ext" size={13} />
          {t("Open {0}", card.entity ? humanizeToolName(card.entity).toLowerCase() : t("record"))}
        </Link>
      )}
    </div>
  );
}

/** A price as a ledger: what priced it, every charge with a bar to its running total, and the limits that held. */
export function DeskRateBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const rate = useMemo(() => rateExplanationFrom(artifact), [artifact]);
  const [rejectedOpen, setRejectedOpen] = useState(false);
  const total = Number(rate.totals.total) || 0;

  return (
    <div className="dk-ax-pad dk-ax-rate">
      {rate.winner && (
        <div className="dk-ax-win">
          <span className="dk-ax-wck">
            <ArtIcon name="check" size={12} stroke={2.6} />
          </span>
          <div>
            <b>{rate.winner.agreementName || rate.winner.agreementCode}</b>
            <span>
              {rate.winner.ruleLabel}
              {rate.winner.agreementCode && (
                <>
                  {" · "}
                  <code>{rate.winner.agreementCode}</code>
                </>
              )}
            </span>
          </div>
        </div>
      )}
      {rate.tieBreak !== "" && <p className="dk-ax-note">{rate.tieBreak}</p>}
      <div className="dk-ax-ledger">
        {rate.components.map((component, index) => {
          const running = Number(component.runningTotal) || 0;
          const amount = Number(component.amount) || 0;
          return (
            <div
              key={`${component.label}-${index}`}
              className={cn("dk-ax-lr", amount < 0 && "dk-neg")}
              style={{ animationDelay: `${index * 60}ms` }}
            >
              <span className="dk-ax-ll">
                <b>{component.label}</b>
                <span>{component.basis}</span>
              </span>
              <span className="dk-ax-bar">
                <i
                  style={{
                    width: `${total > 0 ? Math.max(0, Math.min(1, running / total)) * 100 : 0}%`,
                  }}
                />
              </span>
              <span className="dk-ax-num">{money(component.amount, rate.currency)}</span>
            </div>
          );
        })}
        {rate.guardrails.map((guard) => (
          <div key={guard.kind} className="dk-ax-guard">
            <ArtIcon name="check" size={12} stroke={2.4} />
            {humanizeToolName(guard.kind)} {money(guard.bound, rate.currency)} ·{" "}
            {money(guard.result, rate.currency)}
          </div>
        ))}
        <div className="dk-ax-total">
          <span>{t("Total")}</span>
          <b>{money(rate.totals.total, rate.currency)}</b>
        </div>
      </div>
      {rate.warnings.map((warning) => (
        <div key={warning} className="dk-ax-warn">
          <ArtIcon name="warn" size={13} />
          {warning}
        </div>
      ))}
      {rate.rejected.length > 0 && (
        <>
          <button
            type="button"
            className="dk-ax-more"
            onClick={() => setRejectedOpen((value) => !value)}
          >
            <ArtIcon name={rejectedOpen ? "up" : "down"} size={11} stroke={2.2} />
            {t(
              "{0, plural, one {# agreement didn't apply} other {# agreements didn't apply}}",
              rate.rejected.length,
            )}
          </button>
          {rejectedOpen && (
            <div className="dk-ax-rej">
              {rate.rejected.map((entry) => (
                <div key={entry.agreementCode + entry.ruleLabel}>
                  <code>{entry.agreementCode}</code>
                  <span>{entry.ruleLabel}</span>
                  <em>{entry.detail || entry.reason}</em>
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
}

/** A write-up the agent published, read as it was written. */
export function DeskDocBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const document = useMemo(() => documentFrom(artifact), [artifact]);
  if (document.body.trim() === "") {
    return <Notice>{t("This document is empty.")}</Notice>;
  }
  return (
    <article className="dk-ax-pad dk-ax-doc">
      <h2>{artifact.title}</h2>
      <p className="dk-ax-dm">
        {t(
          "Written {0}",
          new Date(artifact.updatedAt * 1000).toLocaleTimeString([], {
            hour: "numeric",
            minute: "2-digit",
          }),
        )}
      </p>
      <AiMarkdown content={document.body} />
    </article>
  );
}

/** What moved between two runs of a report: the counts, the totals and each change. */
export function DeskDiffBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const diff = useMemo(() => runDiffFrom(artifact), [artifact]);
  const sign: Record<string, string> = { added: "+", removed: "−", changed: "~" };
  const when = (at: number) =>
    at > 0 ? new Date(at * 1000).toLocaleDateString([], { month: "short", day: "numeric" }) : "";

  return (
    <div className="dk-ax-pad dk-ax-diff">
      <div className="dk-ax-dh">
        <span>
          {diff.before.reportName || t("Before")} {when(diff.before.generatedAt)}
        </span>
        <i>→</i>
        <span>
          {diff.after.reportName || t("After")} {when(diff.after.generatedAt)}
        </span>
      </div>
      <div className="dk-ax-dc">
        <span className="dk-add">{t("+{0} added", diff.counts.added)}</span>
        <span className="dk-rem">{t("−{0} removed", diff.counts.removed)}</span>
        <span className="dk-chg">{t("~{0} changed", diff.counts.changed)}</span>
        <span>{t("{0} unchanged", diff.counts.unchanged)}</span>
      </div>
      {diff.totals.length > 0 && (
        <div className="dk-ax-dt">
          {diff.totals.map((measure) => (
            <div key={measure.column}>
              <span>{measure.label}</span>
              <b className="dk-ax-num">{measure.after}</b>
              <em
                className={cn(
                  measure.delta.startsWith("-") || measure.delta.startsWith("−")
                    ? "dk-dn"
                    : measure.delta !== "" && measure.delta !== "0"
                      ? "dk-up"
                      : "",
                )}
              >
                {measure.delta === "" || measure.delta === "0" ? t("no change") : measure.delta}
              </em>
            </div>
          ))}
        </div>
      )}
      <div className="dk-ax-dl">
        {diff.changes.map((change, index) => {
          const measure = change.measures[0];
          return (
            <div
              key={change.key + index}
              className={cn("dk-ax-dr", `dk-${change.kind}`)}
              style={{ animationDelay: `${index * 50}ms` }}
            >
              <span className="dk-ax-dk">{sign[change.kind] ?? "·"}</span>
              <span className="dk-ax-dn">
                <b>{change.keyValues[0] ?? change.key}</b>
                <span>{change.keyValues.slice(1).join(" · ")}</span>
              </span>
              <span className="dk-ax-num dk-mut">{measure?.before || "—"}</span>
              <i>→</i>
              <span className="dk-ax-num">{measure?.after || "—"}</span>
            </div>
          );
        })}
      </div>
      {diff.note !== "" && <p className="dk-ax-note">{diff.note}</p>}
    </div>
  );
}

function stepState(proposal: AssistantProposal | undefined): "done" | "wait" | "next" | "fail" {
  switch (proposal?.status) {
    case "Executed":
    case "Simulated":
      return "done";
    case "Pending":
      return "wait";
    case "Rejected":
    case "ExecutionFailed":
    case "Expired":
      return "fail";
    default:
      return "next";
  }
}

/** A plan as a checklist with its progress, each step ticking as it runs. It is decided in the approval box. */
export function DeskPlanBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const plan = useMemo(() => planFrom(artifact), [artifact]);
  const proposalsQuery = useQuery(queries.assistant.proposals(artifact.threadId));
  const byId = new Map(
    (proposalsQuery.data?.results ?? []).map((proposal) => [proposal.id, proposal]),
  );
  const steps = plan.steps.map((step) => ({
    ...step,
    state: stepState(byId.get(step.proposalId)),
  }));
  const done = steps.filter((step) => step.state === "done").length;
  const total = Math.max(steps.length, plan.stepCount, 1);

  return (
    <div className="dk-ax-pad dk-ax-plan">
      {plan.summary !== "" && <p className="dk-ax-lead">{plan.summary}</p>}
      <div className="dk-ax-prog">
        <span style={{ width: `${(done / total) * 100}%` }} />
      </div>
      <div className="dk-ax-prog-l">{t("{0} of {1} steps done", done, total)}</div>
      <ol>
        {steps.map((step, index) => (
          <li
            key={step.proposalId || step.step}
            className={`dk-st-${step.state}`}
            style={{ animationDelay: `${index * 70}ms` }}
          >
            <span className="dk-ax-sn">
              {step.state === "done" ? <ArtIcon name="check" size={11} stroke={2.8} /> : index + 1}
            </span>
            <div>
              <b>{step.rationale || humanizeToolName(step.toolName)}</b>
              <span>
                <code>{step.toolName}</code>
                {step.state === "wait"
                  ? ` · ${t("waiting on your approval")}`
                  : step.state === "next"
                    ? ` · ${t("up next")}`
                    : step.state === "fail"
                      ? ` · ${t("did not run")}`
                      : ""}
              </span>
            </div>
          </li>
        ))}
      </ol>
      {steps.length === 0 && (
        <p className="dk-ax-note">{t("The plan behind this is no longer in the conversation.")}</p>
      )}
    </div>
  );
}

/** A message waiting to go: who it goes to, what it says and why it is worded so. Sending is decided in the approval box. */
export function DeskEmailBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const draft = useMemo(() => emailDraftFrom(artifact), [artifact]);
  const proposalsQuery = useQuery({
    ...queries.assistant.proposals(artifact.threadId),
    enabled: artifact.proposalId !== "",
  });
  const proposal = proposalsQuery.data?.results.find(
    (candidate) => candidate.id === artifact.proposalId,
  );
  const { copy, isCopied: copied } = useCopyToClipboard();
  const state =
    artifact.status === "Sent" || proposal?.status === "Executed"
      ? "sent"
      : proposal?.status === "Pending"
        ? "waiting"
        : proposal
          ? "decided"
          : "unknown";

  return (
    <div className="dk-ax-mail">
      {draft.to.length > 0 && (
        <div className="dk-ax-mrow">
          <span>{t("To")}</span>
          <div className="dk-ax-to">
            {draft.to.map((to) => (
              <span key={to}>{to}</span>
            ))}
          </div>
        </div>
      )}
      <div className="dk-ax-mrow">
        <span>{t("Subject")}</span>
        <input value={draft.subject} readOnly aria-label={t("Subject")} />
      </div>
      <textarea
        value={
          draft.body ||
          t("The message is written from the organization's template when it is sent.")
        }
        readOnly
        spellCheck={false}
        aria-label={t("Message")}
      />
      {draft.rationale !== "" && (
        <div className="dk-ax-why">
          <b>{t("Why this wording")}</b>
          {draft.rationale}
        </div>
      )}
      <div className="dk-ax-acts">
        <button
          type="button"
          className="dk-ax-btn dk-ghost"
          onClick={() => void copy(`${draft.subject}\n\n${draft.body}`)}
        >
          <ArtIcon name={copied ? "check" : "copy"} size={13} />
          {copied ? t("Copied") : t("Copy")}
        </button>
        <span className="flex-1" />
        {state === "sent" ? (
          <span className="dk-ax-sent">
            <ArtIcon name="check" size={13} stroke={2.4} />
            {t("Sent")}
          </span>
        ) : state === "waiting" ? (
          <span className="dk-ax-sent dk-wait">{t("Waiting on your approval")}</span>
        ) : state === "decided" ? (
          <span className="dk-ax-sent">{t("Decided")}</span>
        ) : null}
      </div>
    </div>
  );
}

/** A view described in words and opened live: what it filters, what it left out, and the way in. */
export function DeskViewBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const view = useMemo(() => composedViewFrom(artifact), [artifact]);
  const navigation = useMemo(() => navigationFrom(artifact), [artifact]);
  if (view) {
    const parts = splitTerms(view.explanation, view.terms);
    return (
      <div className="dk-ax-pad dk-ax-view">
        <p className="dk-ax-vx">
          {parts.map((part, index) =>
            part.term ? (
              <span key={index} className="dk-ax-term">
                {part.text}
              </span>
            ) : (
              part.text
            ),
          )}
        </p>
        <div className="dk-ax-vprev">
          <div className="dk-ax-vbar">
            <span>{humanizeToolName(view.entity)}</span>
            <em>{t("{0, plural, one {# filter} other {# filters}}", view.filterCount)}</em>
          </div>
        </div>
        {view.unresolved.map((entry) => (
          <div key={entry.phrase} className="dk-ax-warn">
            <ArtIcon name="warn" size={13} />
            <span>
              {t("Left out")} <b>“{entry.phrase}”</b>. {entry.reason}
            </span>
          </div>
        ))}
        <Link className="dk-ax-btn dk-ink dk-wide" to={view.path}>
          <ArtIcon name="ext" size={13} />
          {t("Open in {0}", humanizeToolName(view.entity))}
        </Link>
      </div>
    );
  }
  if (navigation) {
    return (
      <div className="dk-ax-pad dk-ax-view">
        <p className="dk-ax-vx">{navigation.location || navigation.name}</p>
        <Link className="dk-ax-btn dk-ink dk-wide" to={navigation.path}>
          <ArtIcon name="ext" size={13} />
          {t("Open {0}", navigation.name)}
        </Link>
      </div>
    );
  }
  return <Notice>{t("This view no longer names a page.")}</Notice>;
}

function splitTerms(
  text: string,
  terms: readonly string[],
): Array<{ text: string; term: boolean }> {
  if (terms.length === 0) {
    return [{ text, term: false }];
  }
  const escaped = terms.map((term) => term.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"));
  const pattern = new RegExp(`(${escaped.join("|")})`, "g");
  return text
    .split(pattern)
    .filter((part) => part !== "")
    .map((part) => ({ text: part, term: terms.includes(part) }));
}

/** A decision the agent asked for: the proposal's record, which follows the decision wherever it is made. */
export function DeskDecisionBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const request = decisionRequestOf({ proposalId: artifact.proposalId, ...artifact.payload });
  if (artifact.proposalId === "" || request === null) {
    return <Notice>{t("This decision no longer names a proposal.")}</Notice>;
  }
  return (
    <div className="dk-ax-pad dk-ax-dec">
      <RequestedDecisionRecords request={request} threadId={artifact.threadId} />
    </div>
  );
}

/** A run the agent started, following the run itself. */
export function DeskReportRunBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const run = useMemo(() => reportRunFrom(artifact), [artifact]);
  if (run === null) {
    return <Notice>{t("This run has no id to follow.")}</Notice>;
  }
  return (
    <div className="dk-ax-pad dk-ax-rep">
      <ReportRunCard run={run} />
    </div>
  );
}
