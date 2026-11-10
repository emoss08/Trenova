import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import type { DeskAttentionItem } from "@/lib/graphql/desk-attention";
import { useT } from "@trenova/shared/i18n/use-t";
import { Link } from "react-router";
import { RunSteps } from "../attention/run-steps";

/**
 * The work agents have in hand right now, the prototype's `.run`: the agent,
 * what it is doing and its run's real progress (reading the record, working
 * out what to do, asking a person to approve). The bar moves only when the
 * run does; nothing here is simulated.
 */
export function NovaWorking({
  items,
  pathOf,
}: {
  items: readonly DeskAttentionItem[];
  /** Where a run opens: the case of the record it works on. */
  pathOf: (item: DeskAttentionItem) => string;
}) {
  const t = useT();

  return (
    <div className="dk-dy-rows">
      {items.map((item) => {
        const work = item.agentWork;
        if (!work) {
          return null;
        }
        return (
          <Link key={item.id} to={pathOf(item)} className="dk-dy-run ui-focus-ring">
            <DeskAgentTile agent={work.agent} size="sm" />
            <span className="dk-dy-run-main">
              <span className="dk-dy-run-ti">{item.title}</span>
              <span className="dk-dy-run-nt">
                {work.note !== "" ? work.note : (work.agent?.name ?? t("An agent"))}
              </span>
            </span>
            <RunSteps work={work} />
          </Link>
        );
      })}
    </div>
  );
}
