import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { AiMarkdown } from "@/components/elements/ai-markdown";
import type {
  WatchtowerAgent,
  WatchtowerAgentStep,
  WatchtowerAgentWork,
  WatchtowerFact,
  WatchtowerItem,
} from "@/lib/graphql/watchtower";
import { withoutRecordIds } from "@/lib/record-ids";
import { agentRunPath } from "@/lib/record-paths";
import { LinkExternal01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useIsTruncated } from "@trenova/shared/hooks/use-is-truncated";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatCompactAge } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { Link } from "react-router";
import { QueueCard } from "../queue/queue";
import type { QueueLeave } from "../queue/use-queue-cursor";
import { AgentRing } from "./agent-ring";
import { DeadlineClock } from "./deadline-clock";
import { WatchtowerKindGlyph } from "./kind-glyph";
import { formatFactTime, minutesLeft } from "./lanes";
import { severityClass, severityLabel } from "./severity";

/** How many facts the record button carries beside the record's name. */
const RECORD_FACTS = 2;

function stepLabel(t: TranslateFn, step: WatchtowerAgentStep, noun: string | null): string {
  switch (step) {
    case "Read":
      return noun ? t("Read the {0}", noun.toLowerCase()) : t("Read the record");
    case "Decide":
      return t("Work out what to do");
    default:
      return t("Ask you to approve");
  }
}

function factText(t: TranslateFn, fact: WatchtowerFact, now: number, timezone: string): string {
  if (fact.value !== null && fact.value !== "") {
    return fact.value;
  }
  return fact.at !== null ? formatFactTime(t, fact.at, now, timezone) : "";
}

function agentLine(t: TranslateFn, work: WatchtowerAgentWork, noun: string | null): string {
  const steps = work.steps.length;
  switch (work.status) {
    case "Working": {
      const at = Math.min(work.step, steps - 1);
      return t(
        "Step {0} of {1} · {2}",
        Math.min(work.step + 1, steps),
        steps,
        stepLabel(t, work.steps[at], noun),
      );
    }
    case "NeedsYou":
      return work.failed ? t("Stopped before it finished") : t("Waiting for you to approve");
    default:
      return t("Done");
  }
}

/**
 * The item the person is working, read whole: how long is left, what kind of
 * thing it is and when it was raised, what is at stake, the record it is
 * about, how it came to this, and who is on it — or who could be.
 */
export function WatchtowerItemCard({
  item,
  timezone,
  leaving,
  onReviewAndApprove,
}: {
  item: WatchtowerItem;
  timezone: string;
  leaving: QueueLeave | null;
  onReviewAndApprove: () => void;
}) {
  const t = useT();
  const now = useNow("second");
  const minutes = minutesLeft(item.dueAt, now);
  const age = now - item.occurredAt;
  const noun = item.record?.noun || null;
  const recordFacts = item.facts
    .slice(0, RECORD_FACTS)
    .map((fact) => {
      const text = factText(t, fact, now, timezone);
      return text === "" ? "" : [fact.label, text].join(" ");
    })
    .filter((text) => text !== "")
    .join(" · ");

  return (
    <QueueCard leaving={leaving} className="dk-w3-card">
      {minutes !== null && <DeadlineClock item={item} minutes={minutes} />}

      <div className="dk-w3-kind">
        <span className={cn("dk-wt-sev", severityClass(item.severity))} aria-hidden />
        <span className="sr-only">{severityLabel(t, item.severity)}</span>
        {item.kindLabel}
        <span>
          {" · "}
          {age < 60 ? t("just now") : t("{0} ago", formatCompactAge(age))}
        </span>
      </div>
      <h1>{item.title}</h1>
      {item.summary !== "" && <p className="dk-dc2-why">{item.summary}</p>}

      {item.impact.length > 0 && (
        <div className="dk-w3-imp">
          <em className="dk-w3-impl">{t("At stake")}</em>
          <ul className="contents">
            {item.impact.map((stake) => (
              <li key={stake}>{stake}</li>
            ))}
          </ul>
        </div>
      )}

      {(item.record !== null || item.path !== "") && (
        <RecordButton item={item} facts={recordFacts} />
      )}

      {item.trail.length > 0 && (
        <ol className="dk-w3-trail" aria-label={t("How it got here")}>
          {item.trail.map((event, index) => (
            <li
              key={`${index}-${event.text}`}
              className={cn(index === item.trail.length - 1 && "dk-last")}
            >
              <i aria-hidden />
              {event.text}
              {event.at !== null && <time>{formatFactTime(t, event.at, now, timezone)}</time>}
            </li>
          ))}
        </ol>
      )}

      {item.agentWork ? (
        <AgentBlock work={item.agentWork} noun={noun} onReviewAndApprove={onReviewAndApprove} />
      ) : (
        item.suggestedAgent && <Suggestion agent={item.suggestedAgent} />
      )}
    </QueueCard>
  );
}

function RecordButton({ item, facts }: { item: WatchtowerItem; facts: string }) {
  const t = useT();
  const body = (
    <>
      <WatchtowerKindGlyph kind={item.sourceKind} className="size-3.5" />
      <span>
        <em>{item.record?.noun || item.kindLabel}</em>
        <b>{item.record?.label || item.title}</b>
      </span>
      <span className="dk-w3-recf">{facts}</span>
      {item.path !== "" && <LinkExternal01Icon aria-hidden className="size-3 flex-none" />}
    </>
  );

  if (item.path === "") {
    return <div className="dk-w3-rec">{body}</div>;
  }

  return (
    <Link className="dk-w3-rec ui-focus-ring" to={item.path} aria-label={t("Open the record")}>
      {body}
    </Link>
  );
}

function AgentBlock({
  work,
  noun,
  onReviewAndApprove,
}: {
  work: WatchtowerAgentWork;
  noun: string | null;
  onReviewAndApprove: () => void;
}) {
  const t = useT();
  const name = work.agent?.name ?? t("An agent");

  return (
    <section className={cn("dk-w3-ag", `dk-st-${work.status}`)} aria-label={t("Agent on it")}>
      <div className="dk-w3-agh">
        <AgentRing work={work} agent={work.agent} size={22} />
        <span>
          <b>{name}</b>
          <em>{agentLine(t, work, noun)}</em>
        </span>
        {work.status === "NeedsYou" &&
          (work.failed ? (
            <Button
              size="sm"
              variant="quiet"
              nativeButton={false}
              render={<Link to={agentRunPath(work.runId)} />}
            >
              {t("See the run")}
            </Button>
          ) : (
            <Button size="sm" onClick={onReviewAndApprove}>
              {t("Review & approve")}
            </Button>
          ))}
      </div>
      {work.note !== "" && <AgentNote key={work.runId} note={work.note} />}
      <div
        className="dk-w3-agbar"
        role="progressbar"
        aria-label={t("Steps")}
        aria-valuemin={0}
        aria-valuemax={work.steps.length}
        aria-valuenow={Math.min(work.step, work.steps.length)}
      >
        {work.steps.map((step, index) => (
          <i
            key={step}
            title={stepLabel(t, step, noun)}
            className={cn(
              index < work.step && "dk-d",
              index === work.step && work.status === "Working" && "dk-n",
            )}
          />
        ))}
      </div>
    </section>
  );
}

/**
 * What the run said, whole: its summary, or why it stopped, in the markdown
 * the model wrote it in, with record ids left out as everywhere a person
 * reads a model. The first lines show at first; when there is more, Show
 * more opens the rest in place.
 */
function AgentNote({ note }: { note: string }) {
  const t = useT();
  const [expanded, setExpanded] = useState(false);
  const [measure, clipped] = useIsTruncated<HTMLDivElement>("block");

  return (
    <div className="dk-w3-agn">
      <div ref={measure} className="dk-w3-agn-body" data-clamped={expanded ? undefined : ""}>
        <AiMarkdown content={withoutRecordIds(note)} className="dk-md" deskSubset />
      </div>
      {(expanded || clipped) && (
        <Button
          size="xs"
          variant="link"
          className="h-auto px-0"
          aria-expanded={expanded}
          onClick={() => setExpanded((open) => !open)}
        >
          {expanded ? t("Show less") : t("Show more")}
        </Button>
      )}
    </div>
  );
}

function Suggestion({ agent }: { agent: WatchtowerAgent }) {
  const t = useT();

  return (
    <div className="dk-w3-sug">
      <DeskAgentTile agent={agent} size="sm" />
      <span>
        <b>{t("{0} can take this", agent.name)}</b>
        {agent.description !== "" && <em>{agent.description}</em>}
      </span>
    </div>
  );
}
