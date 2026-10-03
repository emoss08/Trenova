import { decisionRequestOf } from "@/components/assistant/decision-requests";
import { presentProposal } from "@/components/assistant/proposal-presenters";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { decideMyPlan, decideMyProposal, decideMyProposals } from "@/lib/graphql/agent-decisions";
import { invalidateProposalViews } from "@/lib/proposal-cache";
import { DisplayValue } from "@/components/assistant/display-value";
import {
  formatDisplayValue,
  isDetailType,
  statusPhase,
} from "@/components/assistant/readable-values";
import { ReportRunCard } from "@/components/assistant/report-run-card";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { queries } from "@/lib/queries";
import type { AssistantArtifact, AssistantProposal } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { humanizeToolName } from "@/components/assistant/proposal-state";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import {
  composedViewFrom,
  emailDraftFrom,
  entityCardFrom,
  navigationFrom,
  planFrom,
  rateExplanationFrom,
  reportRunFrom,
  runDiffFrom,
} from "./artifact-payloads";
import { ArtIcon } from "./desk-art-kinds";
import { DeskBillingItem } from "./desk-billing-item";
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

  const viewed = card.view ? (
    <DeskRecordView title={title} view={card.view} entity={card.entity} path={card.path} />
  ) : null;
  // A billing item is reviewed from its card, so it is read live and acted on there.
  if (card.entity === "billing_queue_item" && card.recordId !== "") {
    return (
      <DeskBillingItem
        itemId={card.recordId}
        fallback={viewed ?? <Notice>{t("Loading…")}</Notice>}
      />
    );
  }
  if (viewed) {
    return viewed;
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
      {!rate.winner && rate.pricedBy && (
        <div className="dk-ax-win">
          <span className="dk-ax-wck">
            <ArtIcon name="check" size={12} stroke={2.6} />
          </span>
          <div>
            <b>{rate.pricedBy.method || t("Entered by hand")}</b>
            <span>
              {t("Formula template")}
              {rate.pricedBy.expression && (
                <>
                  {" · "}
                  <code>{rate.pricedBy.expression}</code>
                </>
              )}
            </span>
          </div>
        </div>
      )}
      {!rate.winner && rate.pricedBy?.explanation && (
        <p className="dk-ax-note">{rate.pricedBy.explanation}</p>
      )}
      {rate.pricedBy?.override && (
        <p className="dk-ax-note">
          {t("Overridden to {0}", money(rate.pricedBy.override, rate.currency))}
          {rate.pricedBy.overrideReason ? ` · ${rate.pricedBy.overrideReason}` : ""}
        </p>
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

/**
 * A message waiting to go: who it goes to, what it says and why it is worded
 * so. The subject and body can be changed here; "Send for approval" records
 * the decision on the proposal behind the draft, with the changes as its
 * modifications, the same way the approval box does.
 */
export function DeskEmailBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const queryClient = useQueryClient();
  const draft = useMemo(() => emailDraftFrom(artifact), [artifact]);
  const proposalsQuery = useQuery({
    ...queries.assistant.proposals(artifact.threadId),
    enabled: artifact.proposalId !== "",
  });
  const proposal = proposalsQuery.data?.results.find(
    (candidate) => candidate.id === artifact.proposalId,
  );
  const { copy, isCopied: copied } = useCopyToClipboard();
  const [subject, setSubject] = useState(draft.subject);
  const [body, setBody] = useState(draft.body);
  const subjectKey = typeof artifact.payload.subject === "string" ? "subject" : "";
  const bodyKey = typeof artifact.payload.body === "string" ? "body" : "";
  const state =
    artifact.status === "Sent" || proposal?.status === "Executed"
      ? "sent"
      : proposal?.status === "Pending"
        ? "waiting"
        : proposal
          ? "decided"
          : "unknown";
  const editable = state === "waiting";

  const sendMutation = useApiMutation({
    mutationFn: () => {
      const modifications: Record<string, unknown> = {};
      if (subjectKey && subject !== draft.subject) modifications[subjectKey] = subject;
      if (bodyKey && body !== draft.body) modifications[bodyKey] = body;
      const changed = Object.keys(modifications).length > 0;
      return decideMyProposal(artifact.proposalId, {
        decision: changed ? "Modified" : "Accepted",
        reasonCode: changed ? "modified_from_desk" : "",
        ...(changed ? { modifications } : {}),
      });
    },
    onSuccess: () => invalidateProposalViews(queryClient, artifact.threadId),
    resourceName: "Draft",
  });

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
        <input
          value={subject}
          readOnly={!editable || subjectKey === ""}
          onChange={(event) => setSubject(event.target.value)}
          aria-label={t("Subject")}
        />
      </div>
      <textarea
        value={
          bodyKey === ""
            ? t("The message is written from the organization's template when it is sent.")
            : body
        }
        readOnly={!editable || bodyKey === ""}
        onChange={(event) => setBody(event.target.value)}
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
          onClick={() => void copy(`${subject}\n\n${body}`)}
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
        ) : sendMutation.isSuccess || (state === "decided" && proposal?.status !== "Rejected") ? (
          <span className="dk-ax-sent">
            <ArtIcon name="check" size={13} stroke={2.4} />
            {t("Sent for approval")}
          </span>
        ) : state === "waiting" ? (
          <button
            type="button"
            className="dk-ax-btn dk-ink"
            disabled={sendMutation.isPending || artifact.proposalId === ""}
            onClick={() => sendMutation.mutate()}
          >
            {t("Send for approval")}
          </button>
        ) : state === "decided" ? (
          <span className="dk-ax-sent dk-wait">{t("Set aside")}</span>
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
            {view.count !== null && (
              <b>{t("{0, plural, one {# result} other {# results}}", view.count)}</b>
            )}
          </div>
          {view.preview.map((row) => (
            <div key={row.id} className="dk-ax-vrow">
              <span className="dk-ax-id">{row.id}</span>
              <span>{row.label}</span>
              {row.status !== "" && (
                <span className={cn("dk-ax-pill", PHASE_PILL[statusPhase(row.status) ?? ""])}>
                  <i />
                  {row.status}
                </span>
              )}
            </div>
          ))}
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

/** Where a decision stands, read from the proposals or plan behind it. */
function decisionState(statuses: readonly string[]): "pending" | "approved" | "dismissed" {
  if (statuses.length === 0 || statuses.some((status) => status === "Pending")) {
    return "pending";
  }
  return statuses.some((status) => status === "Rejected" || status === "Expired")
    ? "dismissed"
    : "approved";
}

const DECISION_ROWS = 8;

/**
 * A decision the agent asked for: what it covers, each change, and the way to
 * make it here. Approving and setting aside go through the same decision
 * mutations as the approval box, and the state follows the proposals however
 * they are decided.
 */
export function DeskDecisionBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const queryClient = useQueryClient();
  const request = decisionRequestOf({ proposalId: artifact.proposalId, ...artifact.payload });
  const proposalsQuery = useQuery(queries.assistant.proposals(artifact.threadId));
  const plansQuery = useQuery({
    ...queries.assistant.plans(artifact.threadId),
    enabled: request !== null && request.planId !== "",
  });
  const all = proposalsQuery.data?.results ?? [];
  const plan =
    request?.planId !== ""
      ? (plansQuery.data?.results.find((candidate) => candidate.id === request?.planId) ?? null)
      : null;
  const proposals =
    request === null
      ? []
      : plan !== null
        ? all
            .filter((proposal) => proposal.planId === plan.id)
            .sort((a, b) => a.planStep - b.planStep)
        : request.proposalIds
            .map((id) => all.find((candidate) => candidate.id === id))
            .filter((proposal): proposal is AssistantProposal => proposal !== undefined);
  const state =
    plan !== null
      ? plan.status === "Pending"
        ? "pending"
        : plan.status === "Rejected" || plan.status === "Expired"
          ? "dismissed"
          : "approved"
      : decisionState(proposals.map((proposal) => proposal.status));
  const ids = proposals
    .filter((proposal) => proposal.status === "Pending")
    .map((proposal) => proposal.id);

  const decide = useApiMutation({
    mutationFn: async (decision: "Accepted" | "Rejected") => {
      const input = { decision, reasonCode: decision === "Rejected" ? "not_now" : "" };
      if (plan !== null) {
        await decideMyPlan(plan.id, input);
      } else if (ids.length === 1) {
        await decideMyProposal(ids[0], input);
      } else {
        await decideMyProposals(ids, input);
      }
    },
    onSuccess: () => invalidateProposalViews(queryClient, artifact.threadId),
    resourceName: "Decision",
  });

  if (artifact.proposalId === "" || request === null) {
    return <Notice>{t("This decision no longer names a proposal.")}</Notice>;
  }
  if (proposalsQuery.isPending) {
    return <Notice>{t("Loading…")}</Notice>;
  }

  const first = proposals[0];
  const view = first ? presentProposal(first) : null;
  const scope = [
    view?.summary ?? "",
    first ? humanizeToolName(first.toolName) : "",
    t("{0, plural, one {# change} other {# changes}}", proposals.length),
  ].filter(Boolean);

  return (
    <div className={cn("dk-ax-pad dk-ax-dec", `dk-s-${state}`)}>
      <div className="dk-ax-dech">
        <b>{artifact.title}</b>
        <span className={cn("dk-ax-dst", state !== "pending" && `dk-${state}`)}>
          {state === "pending"
            ? t("Waiting on you")
            : state === "approved"
              ? t("Approved")
              : t("Set aside")}
        </span>
      </div>
      <p className="dk-ax-note">{scope.join(" · ")}</p>
      <div className="dk-ax-decl">
        {proposals.slice(0, DECISION_ROWS).map((proposal) => {
          const shown = presentProposal(proposal);
          const [lead, change] = shown.highlights;
          return (
            <div key={proposal.id}>
              <span className="dk-ax-id">{lead ? String(lead.value) : proposal.id.slice(-8)}</span>
              <span>{shown.title}</span>
              <s>{change ? change.label : ""}</s>
              <i>→</i>
              <em>{change ? String(change.value) : humanizeToolName(proposal.status)}</em>
            </div>
          );
        })}
        {proposals.length > DECISION_ROWS && (
          <div className="dk-ax-decm">{t("+ {0} more", proposals.length - DECISION_ROWS)}</div>
        )}
      </div>
      {state === "pending" && ids.length + (plan !== null ? 1 : 0) > 0 && (
        <div className="dk-ax-acts">
          <button
            type="button"
            className="dk-ax-btn dk-ghost"
            disabled={decide.isPending}
            onClick={() => decide.mutate("Rejected")}
          >
            {t("Not now")}
          </button>
          <span className="flex-1" />
          <button
            type="button"
            className="dk-ax-btn dk-ink"
            disabled={decide.isPending}
            onClick={() => decide.mutate("Accepted")}
          >
            {t("{0, plural, one {Approve # change} other {Approve # changes}}", proposals.length)}
          </button>
        </div>
      )}
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
