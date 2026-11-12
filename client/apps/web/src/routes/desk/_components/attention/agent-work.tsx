import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { AiMarkdown } from "@/components/elements/ai-markdown";
import type { DeskAttentionItem } from "@/lib/graphql/desk-attention";
import { ChevronDownIcon } from "@trenova/shared/components/icons";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Activity, useId, useState } from "react";
import { RunSteps, runDoing } from "./run-steps";
import { workNoteGist, workNoteHasMore } from "./work-note";

type AgentWork = NonNullable<DeskAttentionItem["agentWork"]>;

/** Where the agent stands with the item, as the block's heading says it. */
function standing(t: TranslateFn, work: AgentWork): string {
  const name = work.agent?.name ?? t("An agent");
  if (work.failed) {
    return t("{0} could not finish", name);
  }
  switch (work.status) {
    case "Working":
      return t("{0} is on it", name);
    case "NeedsYou":
      return t("{0} needs your decision", name);
    default:
      return t("{0} looked at this", name);
  }
}

/**
 * The agent on an item, in the least a person needs to act: who it is and
 * where it stands, its real progress while it runs, and one plain sentence of
 * what it found. Its whole report is there for whoever wants it, behind one
 * press, and is not in the way of whoever does not.
 */
export function AgentWorkBlock({ work }: { work: AgentWork }) {
  const t = useT();
  const reportId = useId();
  const [open, setOpen] = useState(false);
  const working = work.status === "Working";
  const gist = working
    ? runDoing(t, work)
    : workNoteGist(work.note) || t("It will ask you before it changes anything.");
  const hasReport = !working && workNoteHasMore(work.note);

  return (
    <section className="mt-5.5">
      <div className="flex items-center gap-2">
        <DeskAgentTile agent={work.agent} size="xs" />
        <h2 className="min-w-0 flex-1 truncate text-sm font-semibold text-dsk-fg">
          {standing(t, work)}
        </h2>
        {work.status !== "Done" && (
          <span className="w-37.5 flex-none">
            <RunSteps work={work} />
          </span>
        )}
      </div>
      <p className="mt-1.5 line-clamp-2 pl-7 text-dsk-base text-dsk-fg2">{gist}</p>
      {hasReport && (
        <div className="pl-7">
          <button
            type="button"
            aria-expanded={open}
            aria-controls={reportId}
            onClick={() => setOpen((shown) => !shown)}
            className="mt-1.5 inline-flex items-center gap-1 rounded-[4px] ui-focus-ring text-dsk-sm text-dsk-subtle hover:text-dsk-fg"
          >
            {open ? t("Hide what it found") : t("What it found")}
            <ChevronDownIcon
              aria-hidden
              className={cn("size-3 transition-transform duration-200", open && "rotate-180")}
            />
          </button>
          {/* Read ahead while hidden, so the report is there the moment it is asked for. */}
          <Activity mode={open ? "visible" : "hidden"}>
            <div
              id={reportId}
              className="mt-2 max-h-80 animate-rise overflow-auto rounded-[8px] border border-dsk-b-sub bg-dsk-card px-3 py-2.5"
            >
              <AiMarkdown content={work.note} className="text-xs text-dsk-fg2" deskSubset />
            </div>
          </Activity>
        </div>
      )}
    </section>
  );
}
