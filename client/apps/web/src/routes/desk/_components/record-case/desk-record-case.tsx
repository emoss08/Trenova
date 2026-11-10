import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { isRecordEntityType, recordPath } from "@/config/record-links";
import { usePermission } from "@/hooks/use-permission";
import type { DeskAttentionItem } from "@/lib/graphql/desk-attention";
import { isTypingTarget } from "@/lib/dom";
import { deskAttention } from "@/lib/queries/desk-attention";
import { useDeskMarginStore } from "@/stores/desk-margin-store";
import { useDeskStore } from "@/stores/desk-store";
import { useDeskSetting } from "@/stores/desk-settings-store";
import { useQuery } from "@tanstack/react-query";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatMinutesSpan, formatUnixTime, resolveUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router";
import { InPlaceRecord, recordCardFor, usePreloadRecordCard } from "../artifacts/cards/registry";
import { casePath } from "../attention/attention-record";
import { dueTone } from "../attention/attention-row";
import { DeclineDialog } from "../attention/decline-dialog";
import { lensLabel } from "../attention/lens";
import { useItemDecision } from "../attention/use-item-decision";
import { useDesk } from "../desk-layout";
import { formatCountdown, formatFactTime, minutesLeft } from "../watchtower/lanes";
import { CaseExhibits } from "./case-exhibits";
import { CasePlan } from "./case-plan";
import { casePageContext, caseSubjectOf } from "./case-subject";

/** The finding's citations, each a numbered exhibit beside it. */
function Cites({
  count,
  lit,
  onLight,
}: {
  count: number;
  lit: number | null;
  onLight: (n: number | null) => void;
}) {
  const t = useT();
  return (
    <>
      {Array.from({ length: count }, (_, index) => index + 1).map((n) => (
        <button
          key={n}
          type="button"
          className={cn("dk-w-cite ui-focus-ring", lit === n && "dk-on")}
          aria-label={t("Exhibit {0}", n)}
          onMouseEnter={() => onLight(n)}
          onMouseLeave={() => onLight(null)}
          onFocus={() => onLight(n)}
          onBlur={() => onLight(null)}
          onClick={() =>
            document.getElementById(`dk-w-exh-${n}`)?.scrollIntoView({ block: "nearest" })
          }
        >
          {n}
        </button>
      ))}
    </>
  );
}

/**
 * A record's case: one record and every open signal about it, read as one
 * decision. Its head says what the record is and how long is left; under it,
 * what the agent found, with each claim numbered to the exhibit it rests on;
 * why it matters and what it proposes; the plan to approve or decline as
 * written; and the record itself. The exhibits run down the side, and the box
 * at the foot asks Nova in the margin with the record attached.
 */
export function DeskRecordCase({ kind, id }: { kind: string; id: string }) {
  const t = useT();
  const navigate = useNavigate();
  const desk = useDesk();
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const now = useNow("minute");
  const { allowed: canWatch } = usePermission(Resource.Watchtower, Operation.Read);
  const { allowed: canDecide } = usePermission(Resource.AgentProposal, Operation.Update);
  const preferredAgent = useDeskSetting("agent");
  const entry = recordCardFor(kind);
  usePreloadRecordCard(entry);

  const attention = useQuery({ ...deskAttention.list({ recordId: id }), enabled: canWatch });
  const group = attention.data?.groups[0] ?? null;
  const items: DeskAttentionItem[] = group?.items ?? [];
  const record = group?.record ?? null;
  const label = record?.readable && record.label !== "" ? record.label : id;
  const decisionItem = items.find((item) => item.deskLens === "Decisions") ?? null;
  const decision = useItemDecision(decisionItem);
  const finder = decisionItem?.agentWork ?? items.find((item) => item.agentWork)?.agentWork ?? null;
  const condition = items.find((item) => item.deskLens === "Watchtower") ?? null;
  const [lit, setLit] = useState<number | null>(null);
  const [declining, setDeclining] = useState(false);
  const minutes = minutesLeft(group?.dueAt, now);
  const countdown = formatCountdown(t, minutes);
  const pageLink = isRecordEntityType(kind) ? recordPath(kind, id) : null;

  // The case is what ⌘J and Ask take with them, and it joins Open work.
  const setPageSubject = useDeskMarginStore((state) => state.setPageSubject);
  const noteOpenWork = useDeskMarginStore((state) => state.noteOpenWork);
  useEffect(() => {
    const subject = { kind, id, label };
    setPageSubject(subject);
    if (record?.readable) {
      noteOpenWork(subject);
    }
  }, [id, kind, label, noteOpenWork, record?.readable, setPageSubject]);
  useEffect(() => () => setPageSubject(null), [setPageSubject]);

  // ⌘Enter approves the plan in front of the person; D declines it.
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.defaultPrevented || declining || decision.phase !== "idle" || !canDecide) {
        return;
      }
      if (decision.proposals.length === 0) {
        return;
      }
      const unseen = decision.tally.loading + decision.tally.failed + decision.tally.stale;
      if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
        const box = event.target instanceof HTMLTextAreaElement ? event.target : null;
        if (box && box.value.trim() !== "") {
          return;
        }
        event.preventDefault();
        if (unseen === 0) {
          void decision.approve();
        }
        return;
      }
      if (
        event.key.toLowerCase() === "d" &&
        !event.metaKey &&
        !event.ctrlKey &&
        !event.altKey &&
        !isTypingTarget(event.target)
      ) {
        event.preventDefault();
        setDeclining(true);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  const ask = (question: string) => {
    const subject = { kind, id, label };
    const caseType = caseSubjectOf(kind);
    const existing =
      caseType === null
        ? null
        : (desk.threads.find(
            (thread) => thread.subjectType === caseType && thread.subjectId === id,
          ) ?? null);
    const margin = useDeskMarginStore.getState();
    const open = margin.margin?.threadId ?? existing?.id ?? null;
    if (open !== null) {
      useDeskStore.getState().askAgent(open, question);
      margin.showThread(open, subject);
      return;
    }
    const agent = desk.agents.find((choice) => choice.id === preferredAgent) ?? desk.agents[0];
    if (!agent) {
      margin.openMargin(subject);
      return;
    }
    margin.openMargin(subject);
    desk.startInMargin(
      agent.id,
      question,
      { files: [], providerId: "", mentions: [{ type: kind, id, label }] },
      {
        title: label,
        subjectType: caseType ?? undefined,
        subjectId: caseType ? id : undefined,
        page: casePageContext(casePath(kind, id), kind, id, label),
      },
    );
  };

  const raisedAt = condition ? formatFactTime(t, condition.occurredAt, now, timezone) : "";
  const trust = [
    condition && condition.impact.length > 0
      ? { key: "matters", label: t("Why it matters"), value: condition.impact.join(" · ") }
      : null,
    countdown !== null && minutes !== null && group?.dueLabel
      ? {
          key: "nothing",
          label: t("If nothing changes"),
          value:
            minutes >= 0
              ? t("{0} in {1}.", group.dueLabel, countdown)
              : t("{0}: already past, {1} ago.", group.dueLabel, formatMinutesSpan(-minutes)),
        }
      : null,
    decisionItem && finder?.agent
      ? { key: "proposes", label: t("{0} proposes", finder.agent.name), value: decisionItem.title }
      : null,
    decisionItem
      ? {
          key: "why",
          label: t("Needs your approval because"),
          value: t("AI control lets this agent only propose this change; a person approves it."),
        }
      : null,
  ].filter((cell): cell is { key: string; label: string; value: string } => cell !== null);

  return (
    <div className="dk-w-case-scroll">
      <div className="dk-w-case">
        <div className="dk-w-case-main">
          <div className="dk-w-chead">
            <div>
              <div className="dk-w-chead-k">
                <span>{record?.noun ?? kind}</span>
                <span>·</span>
                <span className="dk-w-mono">{label}</span>
                {record?.readable && record.status !== "" && (
                  <>
                    <span>·</span>
                    <span>{record.status}</span>
                  </>
                )}
              </div>
              <h1>{record?.readable && record.subtitle !== "" ? record.subtitle : label}</h1>
              <div className="dk-w-chead-sub">
                {group ? (
                  <span
                    className={cn(
                      "dk-w-bdg",
                      group.severity === "Critical"
                        ? "dk-crit"
                        : group.severity === "Warning"
                          ? "dk-warn"
                          : "dk-blue",
                    )}
                  >
                    {t("{0, plural, one {# thing open} other {# things open}}", items.length)}
                  </span>
                ) : (
                  attention.isSuccess && <span className="dk-w-bdg dk-ok">{t("Nothing open")}</span>
                )}
                <span>
                  {items
                    .map((item) => lensLabel(t, item.deskLens))
                    .filter((lens, index, all) => all.indexOf(lens) === index)
                    .join(" · ")}
                </span>
                {pageLink && (
                  <Link className="dk-w-alnk ui-focus-ring" to={pageLink}>
                    {t("Open the record")}
                  </Link>
                )}
              </div>
            </div>
            {countdown !== null && (
              <div className="dk-w-chead-clk">
                <div className={cn("dk-w-chead-v", dueTone(minutes))}>{countdown}</div>
                <div className="dk-w-chead-l">{group?.dueLabel || t("until its deadline")}</div>
              </div>
            )}
          </div>

          {attention.isPending && canWatch ? (
            <div className="dk-w-sk" aria-busy aria-label={t("Reading the case")}>
              <i className="dk-sk" />
              <i className="dk-sk" />
            </div>
          ) : items.length === 0 ? (
            <div className="dk-w-find">
              <div className="dk-w-banner">
                <DeskIcon name="check" size={14} />
                <span>
                  {t("Nothing is open about this record. Ask Nova below if something looks wrong.")}
                </span>
              </div>
            </div>
          ) : finder?.agent && decisionItem ? (
            <div className="dk-w-find">
              <div className="dk-w-find-who">
                <NovaBadge size="lg" />
                <b>{finder.agent.name}</b>
                {t(
                  "found this at {0} from {1, plural, one {# source} other {# sources}}",
                  formatUnixTime(decisionItem.occurredAt, { timezone }),
                  items.length,
                )}
              </div>
              <p className="dk-w-verdict">
                {decision.proposals[0]?.rationale || decisionItem.summary || decisionItem.title}
                <Cites count={items.length} lit={lit} onLight={setLit} />
              </p>
              {trust.length > 0 && (
                <div className="dk-w-trust">
                  {trust.map((cell) => (
                    <div key={cell.key}>
                      <span className="dk-w-trust-k">{cell.label}</span>
                      {cell.value}
                    </div>
                  ))}
                </div>
              )}
            </div>
          ) : (
            <div className="dk-w-find">
              <div className="dk-w-banner">
                <DeskIcon name="radar" size={14} />
                <span>
                  {condition ? (
                    <>
                      <b className="font-medium">{t("Watchtower raised this {0}.", raisedAt)}</b>{" "}
                      {condition.summary || condition.title}
                    </>
                  ) : (
                    items[0]?.summary || items[0]?.title
                  )}
                  <Cites count={items.length} lit={lit} onLight={setLit} />
                </span>
              </div>
              {trust.length > 0 && (
                <div className="dk-w-trust">
                  {trust.map((cell) => (
                    <div key={cell.key}>
                      <span className="dk-w-trust-k">{cell.label}</span>
                      {cell.value}
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {decisionItem && (
            <CasePlan
              item={decisionItem}
              decision={decision}
              canDecide={canDecide}
              timezone={timezone}
              onDecline={() => setDeclining(true)}
            />
          )}

          {entry && (
            <section aria-label={t("The record")} className="dk-w-case-rec">
              <div className="dk-w-sh">
                <h2>{t("The record")}</h2>
              </div>
              <InPlaceRecord
                entry={entry}
                recordId={id}
                order={[id]}
                onSelect={() => undefined}
                onBack={() => void navigate("/desk/needs")}
                backLabel={t("Needs you")}
              />
            </section>
          )}
        </div>

        <CaseExhibits items={items} lit={lit} now={now} timezone={timezone} />
      </div>

      <CaseDock label={label} onAsk={ask} />

      <DeclineDialog
        open={declining}
        agentName={finder?.agent?.name ?? null}
        pending={decision.phase === "deciding"}
        onOpenChange={setDeclining}
        onDecline={async (reason) => {
          await decision.decline(reason);
          setDeclining(false);
        }}
      />
    </div>
  );
}

/**
 * The box at the foot of the case: a few questions worth asking about it and
 * a line to ask your own. The answer opens in Nova's margin, beside the case,
 * with the record attached.
 */
function CaseDock({ label, onAsk }: { label: string; onAsk: (question: string) => void }) {
  const t = useT();
  const [draft, setDraft] = useState("");
  const starters = [
    t("What happens if nothing changes?"),
    t("What are my options here?"),
    t("Draft a note to the customer about this"),
  ];
  const send = (question: string) => {
    const trimmed = question.trim();
    if (trimmed === "") {
      return;
    }
    setDraft("");
    onAsk(trimmed);
  };

  return (
    <div className="dk-w-dock">
      <div className="dk-w-chips" aria-label={t("Suggested questions")}>
        {starters.map((question) => (
          <button
            key={question}
            type="button"
            className="dk-w-chip ui-focus-ring"
            onClick={() => send(question)}
          >
            {question}
          </button>
        ))}
      </div>
      <div className="dk-w-cmp">
        <label className="sr-only" htmlFor="dk-w-case-ask">
          {t("Ask about {0}", label)}
        </label>
        <textarea
          id="dk-w-case-ask"
          rows={1}
          value={draft}
          placeholder={t("Ask about {0}, or tell the agent what to change…", label)}
          onChange={(event) => setDraft(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter" && !event.shiftKey && !event.metaKey && !event.ctrlKey) {
              event.preventDefault();
              send(draft);
            }
            if (event.key === "Escape") {
              event.currentTarget.blur();
            }
          }}
        />
        <div className="dk-w-cmp-row">
          <span className="dk-w-cmp-ctx" title={t("Shared with the question")}>
            <DeskIcon name="pin" size={11} />
            {label}
          </span>
          <span className="dk-w-sp" />
          <button
            type="button"
            className={cn("dk-w-send ui-focus-ring", draft.trim() !== "" && "dk-on")}
            aria-label={t("Send")}
            onClick={() => send(draft)}
          >
            <DeskIcon name="up" size={14} />
          </button>
        </div>
      </div>
    </div>
  );
}
