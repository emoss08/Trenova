import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import type { WorkingRun } from "@/lib/graphql/ai-control";
import type { DeskAttentionGroup, DeskAttentionItem } from "@/lib/graphql/desk-attention";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { Link } from "react-router";
import { agoOrIn } from "../attention/panels/panel-time";
import { RunSteps, runDoing } from "../attention/run-steps";

/**
 * The items an agent's run is working on at this moment. A run that finished,
 * failed or is waiting on a person is not working: what it found is under
 * Needs you or Agents handled, never here.
 */
export function workingItems(groups: readonly DeskAttentionGroup[]): DeskAttentionItem[] {
  return groups
    .flatMap((group) => group.items)
    .filter((item) => item.agentWork?.status === "Working");
}

/** One line of what agents are working on: an agent, and the run it has going. */
export type WorkingRow = {
  key: string;
  /** The agent whose run it is, as the run names it; null for one deleted since. */
  agent: WorkingRun["agent"] | NonNullable<DeskAttentionItem["agentWork"]>["agent"];
  /** The item the run is working, when it is working one of Needs you's. */
  item: DeskAttentionItem | null;
  /** The run itself, for a reader who may read runs. */
  run: WorkingRun | null;
  to: string;
};

/** Where a run that is on no item of Needs you is read: its page in AI control. */
function runPath(runId: string): string {
  return `/admin/agent-control?tab=activity&view=runs&run=${encodeURIComponent(runId)}`;
}

/**
 * What agents are working on, one row a run. The runs are the truth, the same
 * ones AI control lists as working, so the two always agree; a run that is
 * working an item of Needs you is shown as that item and opens its case.
 *
 * The agent is the run's own, never looked up among the agents the reader
 * may ask: most runs here are of agents that run by themselves, which nobody
 * asks. A reader who may not read runs is given the items alone, which is all
 * of the runs they could open anyway.
 */
export function workingRows(
  runs: readonly WorkingRun[] | null,
  items: readonly DeskAttentionItem[],
  pathOf: (item: DeskAttentionItem) => string,
): WorkingRow[] {
  const ofItem = (item: DeskAttentionItem, run: WorkingRun | null): WorkingRow => ({
    key: item.id,
    agent: item.agentWork?.agent ?? run?.agent ?? null,
    item,
    run,
    to: pathOf(item),
  });
  if (runs === null) {
    return items.map((item) => ofItem(item, null));
  }

  const itemOfRun = new Map(items.map((item) => [item.agentWork?.runId ?? "", item] as const));

  return runs.map((run) => {
    const item = itemOfRun.get(run.id);
    if (item) {
      return ofItem(item, run);
    }

    return { key: run.id, agent: run.agent, item: null, run, to: runPath(run.id) };
  });
}

/**
 * What a run on no item of Needs you is on: what it has said of itself, else
 * the record it is on by name. An organization-wide run, a sweep or a digest,
 * has no record and says nothing until it ends, so it has no subject and its
 * row leads with the agent.
 */
export function runSubject(run: WorkingRun): string {
  return run.summary.trim() || run.subjectLabel;
}

/** How a run came to be going, said beside what it is doing. */
export function runOrigin(t: TranslateFn, run: WorkingRun): string {
  if (run.subjectType === "AssistantThread") {
    return t("Answering in a conversation");
  }
  switch (run.trigger) {
    case "Scheduled":
      return t("On its schedule");
    case "Event":
      return t("Started by an event");
    case "Continuous":
      return t("Watching continuously");
    case "Wait":
      return t("Picking up what it was waiting on");
    case "Chat":
      return t("Answering in a conversation");
    default:
      return t("Started by a person");
  }
}

function runWords(t: TranslateFn, status: WorkingRun["status"] | undefined): string {
  switch (status) {
    case "Diagnosing":
      return t("Working out what to do");
    case "GatheringContext":
      return t("Reading the record");
    default:
      return t("Starting");
  }
}

/**
 * The work agents have in hand right now: one line a run, with the agent's
 * own mark and name, what the run is on and what it is doing. A run on an
 * item shows its real progress, any other how long it has been going. The
 * bar moves only when the run does; nothing here is simulated.
 */
export function AgentsWorking({ rows }: { rows: readonly WorkingRow[] }) {
  const t = useT();
  const now = useNowSeconds(30_000);

  return (
    <div className="flex flex-col">
      {rows.map((row) => {
        const work = row.item?.agentWork ?? null;
        const name = row.agent?.name ?? t("An agent");
        const doing = work ? runDoing(t, work) : runWords(t, row.run?.status);
        const subject = row.item?.title ?? (row.run ? runSubject(row.run) : "");
        // A run on nothing a reader could name leads with its agent instead.
        const lead = subject === "" && row.run ? runOrigin(t, row.run) : name;
        const began = row.run ? (row.run.startedAt ?? row.run.createdAt) : 0;

        return (
          <Link
            key={row.key}
            to={row.to}
            className="grid grid-cols-[24px_minmax(0,1fr)_150px] items-center gap-3 rounded-[8px] ui-focus-ring px-3 py-1.5 hover:bg-dsk-hover @max-[560px]/mainc:grid-cols-[24px_minmax(0,1fr)]"
          >
            <DeskAgentTile agent={row.agent} size="sm" />
            <span className="flex min-w-0 flex-col">
              <span className="truncate text-dsk-base text-dsk-fg">{subject || name}</span>
              <span className="truncate text-dsk-sm text-dsk-subtle">
                {subject === "" && !row.run ? doing : t("{0} · {1}", lead, doing)}
              </span>
            </span>
            <span className="text-right text-dsk-sm text-dsk-subtle tabular-nums @max-[560px]/mainc:hidden">
              {work ? <RunSteps work={work} /> : agoOrIn(t, began, now)}
            </span>
          </Link>
        );
      })}
    </div>
  );
}
