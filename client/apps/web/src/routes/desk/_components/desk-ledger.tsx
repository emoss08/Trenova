import { FeedbackControl } from "@/components/ai-feedback/feedback-control";
import { AgentTile } from "@/components/agent-identity/agent-tile";
import { WorkingDot } from "@/components/assistant/voice/working-dot";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { conversationPath } from "@/lib/conversation-path";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { Briefing, BriefingSection } from "@/lib/graphql/briefing";
import type { WatchtowerItem } from "@/lib/graphql/watchtower";
import type { AssistantLiveTurn } from "@/types/assistant";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useCountUp } from "@trenova/shared/hooks/use-count-up";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatShortAge } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowUpRightIcon } from "lucide-react";
import type { CSSProperties, ReactNode } from "react";
import { Link } from "react-router";
import type { usePendingDecisionSummary } from "./decisions/use-pending-decisions";

type PendingDecisionSummary = NonNullable<ReturnType<typeof usePendingDecisionSummary>["data"]>;

/** Watchtower items the ledger names before the rest are counted. */
export const LEDGER_WATCHTOWER_ROWS = 3;
/** Agents named under the waiting count before the rest are counted. */
const NAMED_AGENTS = 3;

export type DeskLedgerProps = {
  now: number;
  agentsById: ReadonlyMap<string, AgentChoice>;
  /** Each conversation's title and agent, for the replies under way. */
  threads: ReadonlyMap<string, { title: string; agentDefinitionId: string }>;
  canDecide: boolean;
  canWatch: boolean;
  waiting: number;
  summary: { pending: boolean; data: PendingDecisionSummary | null | undefined };
  live: { pending: boolean; items: readonly AssistantLiveTurn[] };
  watchtower: {
    pending: boolean;
    items: readonly WatchtowerItem[];
    unseen: number;
    critical: number;
  };
  briefing: Briefing | null;
  /** Staggers each part in behind the one above it, in reading order. */
  entrance: (step: number) => CSSProperties;
  className?: string;
  style?: CSSProperties;
};

/**
 * The desk itself, under the question: the day's figures on one line and
 * one ledger of what is on it.
 *
 * It is one frame and one kind of row, whatever the row is about. A
 * decision waiting, an agent writing, a thing the watchtower saw and a line
 * of the morning's briefing all take the same shape — a mark, a title, a
 * quiet note, a figure or an age at the right edge — in the same rhythm,
 * under the same small label. The figures above it count up as they land,
 * and every one of them opens the page that holds its rows.
 */
export function DeskLedger({
  now,
  agentsById,
  threads,
  canDecide,
  canWatch,
  waiting,
  summary,
  live,
  watchtower,
  briefing,
  entrance,
  className,
  style,
}: DeskLedgerProps) {
  const t = useT();
  const sections = briefing?.sections.filter(
    (section) => section.read !== "" || section.summary !== "" || section.items.length > 0,
  );
  const hasDay = Boolean(briefing && sections && sections.length > 0);

  return (
    <div className={cn("flex flex-col gap-4", className)} style={style}>
      <KpiStrip
        aria-label={t("Today in figures")}
        minItemWidth="9rem"
        className="ring-foreground/10 border-0 ring-1"
      >
        {canDecide && (
          <KpiStripItem
            label={t("Waiting on you")}
            value={<Figure value={waiting} />}
            sub={
              summary.data?.oldestAt
                ? t("Oldest {0}", formatShortAge(now - summary.data.oldestAt))
                : t("Decisions")
            }
            to="/desk/decisions"
          />
        )}
        <KpiStripItem
          label={t("Replying now")}
          value={<Figure value={live.items.length} />}
          sub={t("Agents writing")}
        />
        {canWatch && (
          <KpiStripItem
            label={t("Watchtower")}
            value={<Figure value={watchtower.unseen} />}
            tone={watchtower.critical > 0 ? "danger" : undefined}
            sub={
              watchtower.critical > 0
                ? t("{0, plural, one {One critical} other {# critical}}", watchtower.critical)
                : t("Unseen")
            }
            to="/desk/watchtower"
          />
        )}
      </KpiStrip>

      <div
        style={entrance(1)}
        className={cn(
          "animate-rise bg-card ring-foreground/10 overflow-hidden rounded-lg ring-1",
          hasDay && "xl:grid xl:grid-cols-2",
        )}
      >
        <div className="divide-border-subtle divide-y">
          {canDecide && (
            <LedgerGroup
              label={t("Waiting on you")}
              count={waiting}
              to="/desk/decisions"
              action={t("Review decisions")}
              pending={summary.pending}
              empty={t("Nothing is waiting on you.")}
            >
              {(summary.data?.byAgent ?? []).slice(0, NAMED_AGENTS).map((row) => {
                const agent = agentsById.get(row.agentDefinitionId) ?? {
                  id: row.agentDefinitionId,
                  name: row.agentName,
                };
                return (
                  <LedgerRow
                    key={row.agentDefinitionId}
                    to="/desk/decisions"
                    mark={<AgentTile agent={agent} size="xs" />}
                    title={row.agentName || t("Retired agent")}
                    note={t(
                      "{0, plural, one {One proposal to decide} other {# proposals to decide}}",
                      row.count,
                    )}
                    trailing={<span className="tabular-nums">{row.count}</span>}
                  />
                );
              })}
              {(summary.data?.byAgent.length ?? 0) > NAMED_AGENTS && (
                <LedgerMore>
                  {t(
                    "{0, plural, one {and one more agent} other {and # more agents}}",
                    (summary.data?.byAgent.length ?? 0) - NAMED_AGENTS,
                  )}
                </LedgerMore>
              )}
            </LedgerGroup>
          )}

          <LedgerGroup
            label={t("Replying now")}
            count={live.items.length}
            pending={live.pending}
            empty={t("No agent is writing a reply right now.")}
          >
            {live.items.map((turn) => {
              const thread = threads.get(turn.threadId);
              const agent = thread ? (agentsById.get(thread.agentDefinitionId) ?? null) : null;
              return (
                <LedgerRow
                  key={turn.turnId}
                  to={conversationPath(turn.threadId)}
                  mark={<WorkingDot working still />}
                  title={turn.threadTitle || thread?.title || t("Untitled conversation")}
                  note={agent?.name ?? t("Agent")}
                  trailing={formatShortAge(now - turn.startedAt)}
                />
              );
            })}
          </LedgerGroup>

          {canWatch && (
            <LedgerGroup
              label={t("Watchtower")}
              count={watchtower.unseen}
              tone={watchtower.critical > 0 ? "danger" : undefined}
              to="/desk/watchtower"
              action={t("Open the watchtower")}
              pending={watchtower.pending}
              empty={t("Nothing unresolved. The watchtower is quiet.")}
            >
              {watchtower.items.slice(0, LEDGER_WATCHTOWER_ROWS).map((item) => (
                <LedgerRow
                  key={item.id}
                  to={item.path || "/desk/watchtower"}
                  mark={<SeverityDot severity={item.severity} />}
                  title={item.title}
                  note={item.kindLabel}
                  trailing={formatShortAge(now - item.occurredAt)}
                />
              ))}
            </LedgerGroup>
          )}
        </div>

        {briefing && sections && hasDay && (
          <div className="divide-border-subtle border-border-subtle divide-y border-t xl:border-t-0 xl:border-l">
            <LedgerGroup
              label={t("Your day")}
              aside={
                <>
                  {briefing.narrated && (
                    <AssistMark
                      className="text-foreground-subtle size-3"
                      aria-label={t(
                        "Some of this wording was written by a model and checked against the figures",
                      )}
                    />
                  )}
                  <FeedbackControl target={{ targetType: "Briefing", targetId: briefing.id }} />
                </>
              }
            >
              {sections.map((section) => (
                <BriefingRow key={section.key} briefingId={briefing.id} section={section} />
              ))}
            </LedgerGroup>
          </div>
        )}
      </div>
    </div>
  );
}

/** A figure that counts up to its value as it lands. */
function Figure({ value }: { value: number }) {
  const shown = useCountUp(value);
  return <>{shown > 999 ? "999+" : shown}</>;
}

function LedgerGroup({
  label,
  count,
  tone,
  to,
  action,
  aside,
  pending = false,
  empty,
  children,
}: {
  label: string;
  count?: number;
  tone?: "danger";
  to?: string;
  action?: string;
  aside?: ReactNode;
  pending?: boolean;
  empty?: string;
  children: ReactNode;
}) {
  const rows = Array.isArray(children) ? children.filter(Boolean) : children ? [children] : [];
  const bare = !pending && rows.length === 0;

  return (
    <section aria-label={label} className="group/ledger flex flex-col py-1.5">
      <header className="flex h-8 items-center gap-2 px-4">
        <h2 className="text-foreground-subtle min-w-0 flex-1 truncate text-xs font-medium">
          {label}
        </h2>
        {aside}
        {count !== undefined && (
          <span
            className={cn(
              "text-xs tabular-nums",
              tone === "danger" ? "text-danger-foreground" : "text-foreground-subtle",
            )}
          >
            {count > 999 ? "999+" : count}
          </span>
        )}
        {to && action && (
          <Link
            to={to}
            aria-label={action}
            title={action}
            className="ui-focus-ring text-foreground-subtle hover:text-foreground flex size-6 items-center justify-center rounded-md transition-colors"
          >
            <ArrowUpRightIcon className="size-3.5" />
          </Link>
        )}
      </header>
      {pending ? (
        <LedgerSkeleton />
      ) : bare ? (
        <p className="text-foreground-subtle px-4 pt-1 pb-2 text-sm">{empty}</p>
      ) : (
        <ul className="flex flex-col">{rows}</ul>
      )}
    </section>
  );
}

function LedgerRow({
  to,
  mark,
  title,
  note,
  trailing,
  aside,
}: {
  to?: string;
  mark: ReactNode;
  title: string;
  note?: ReactNode;
  trailing?: ReactNode;
  aside?: ReactNode;
}) {
  const body = (
    <>
      <span className="flex w-4 shrink-0 items-center justify-center">{mark}</span>
      <span className="grid min-w-0 flex-1 leading-tight">
        <span className="truncate text-sm">{title}</span>
        {note && <span className="text-foreground-subtle truncate text-xs">{note}</span>}
      </span>
      {aside}
      {trailing !== undefined && (
        <span className="text-foreground-subtle shrink-0 text-xs tabular-nums">{trailing}</span>
      )}
    </>
  );
  const rowClass = "flex min-h-10 items-center gap-3 px-4 py-1.5";

  return (
    <li className="group/row animate-land">
      {to ? (
        <Link
          to={to}
          className={cn(rowClass, "ui-inset-focus-ring hover:bg-surface-hover transition-colors")}
        >
          {body}
        </Link>
      ) : (
        <div className={rowClass}>{body}</div>
      )}
    </li>
  );
}

function LedgerMore({ children }: { children: ReactNode }) {
  return <li className="text-foreground-subtle px-4 pt-0.5 pb-1.5 pl-11 text-xs">{children}</li>;
}

function LedgerSkeleton() {
  return (
    <div className="flex flex-col gap-2 px-4 pt-1 pb-2" aria-busy>
      <Skeleton className="h-4 w-4/5" />
      <Skeleton className="h-4 w-3/5" />
    </div>
  );
}

function SeverityDot({ severity }: { severity: WatchtowerItem["severity"] }) {
  return (
    <span
      aria-hidden
      className={cn(
        "size-1.5 rounded-full",
        severity === "Critical"
          ? "bg-danger"
          : severity === "Warning"
            ? "bg-warning"
            : "bg-foreground-subtle",
      )}
    />
  );
}

/**
 * One section of the morning's briefing as a row of the ledger: its title,
 * the sentence that was written for it, and its figures at the right edge,
 * each a link to the rows it counted, so a number nobody can trace does not
 * survive this screen.
 */
function BriefingRow({ briefingId, section }: { briefingId: string; section: BriefingSection }) {
  const note = section.read || section.summary;
  return (
    <LedgerRow
      to={section.path || undefined}
      mark={<span aria-hidden className="bg-foreground-subtle/60 size-1.5 rounded-full" />}
      title={section.title}
      note={note || undefined}
      aside={
        <span className="flex shrink-0 items-center">
          <FeedbackControl
            target={{
              targetType: "BriefingSection",
              targetId: briefingId,
              targetPart: section.key,
            }}
            revealClassName="opacity-0 group-hover/row:opacity-100 focus-within:opacity-100"
          />
        </span>
      }
      trailing={
        section.items.length > 0 ? (
          <span className="flex items-baseline gap-3">
            {section.items.map((item) => (
              <span key={item.label} className="flex items-baseline gap-1">
                <span className="text-foreground-subtle">{item.label}</span>
                <span className="text-foreground text-sm">{item.value}</span>
              </span>
            ))}
          </span>
        ) : undefined
      }
    />
  );
}
