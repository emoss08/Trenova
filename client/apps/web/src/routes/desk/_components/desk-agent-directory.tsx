import { AgentTile } from "@/components/agent-identity/agent-tile";
import { useAgentChoices } from "@/components/assistant/use-agent-choices";
import type { AgentChoice, AgentOrigin } from "@/lib/graphql/agent-definition";
import type { AgentRecency } from "@/lib/recent-agents";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowRightIcon, SearchIcon, XIcon } from "lucide-react";
import { useState } from "react";

/** Cards shown before "Show more", and how many more each press reveals. */
const FIRST_SHOWING = 6;
const SHOW_MORE_STEP = 12;
/** Below this many agents the search and the filter are noise, so they stay out of the way. */
const CONTROLS_FROM = FIRST_SHOWING + 1;

const nowInSeconds = () => Math.floor(Date.now() / 1000);

export type DeskAgentDirectoryProps = {
  recency: AgentRecency;
  /** The agent the ask box is on, marked so the two read as one choice. */
  selectedId: string | null;
  disabled?: boolean;
  onChoose: (agent: AgentChoice) => void;
  className?: string;
};

/**
 * Every agent the person can ask, a page at a time.
 *
 * This replaced a grid that drew every agent the organization had. At six
 * that is a directory; at a hundred it is the whole page. So the list is the
 * server's — searched there over names and descriptions, filtered there by
 * where an agent came from, read by cursor — and the page shows a handful,
 * the person's own first, with "Show more" for the rest.
 *
 * Choosing a card does not open a conversation. It points the ask box at
 * that agent, whose questions then appear under it, because the next thing
 * a person does after choosing who to ask is ask.
 */
export function DeskAgentDirectory({
  recency,
  selectedId,
  disabled = false,
  onChoose,
  className,
}: DeskAgentDirectoryProps) {
  const t = useT();
  const [now] = useState(nowInSeconds);
  const [search, setSearch] = useState("");
  const [origin, setOrigin] = useState<AgentOrigin>("all");
  const [showing, setShowing] = useState(FIRST_SHOWING);
  const choices = useAgentChoices({ search, origin, recentIds: recency.ids });

  const agents = [...choices.recent, ...choices.items];
  const visible = agents.slice(0, showing);
  const narrowed = choices.settledSearch !== "" || origin !== "all";
  const total = choices.totalCount;
  const hasMore = agents.length > showing || choices.hasNextPage;
  const showControls = narrowed || search !== "" || (total ?? 0) >= CONTROLS_FROM;

  const narrow = (next: { search?: string; origin?: AgentOrigin }) => {
    if (next.search !== undefined) {
      setSearch(next.search);
    }
    if (next.origin !== undefined) {
      setOrigin(next.origin);
    }
    setShowing(FIRST_SHOWING);
  };

  const showMore = () => {
    const next = showing + SHOW_MORE_STEP;
    setShowing(next);
    if (next > agents.length) {
      choices.fetchNextPage();
    }
  };

  if (!narrowed && !choices.isLoading && (total ?? agents.length) <= 1) {
    return null;
  }

  return (
    <section aria-labelledby="desk-agents-heading" className={cn("flex flex-col gap-3", className)}>
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
        <h2 id="desk-agents-heading" className="flex items-baseline gap-1.5 text-sm font-semibold">
          {t("Agents")}
          {total !== null && (
            <span className="text-muted-foreground text-xs font-normal tabular-nums">{total}</span>
          )}
        </h2>
        {showControls && (
          <div className="ml-auto flex min-w-0 flex-wrap items-center gap-2">
            <Input
              value={search}
              onChange={(event) => narrow({ search: event.target.value })}
              placeholder={t("Search agents")}
              aria-label={t("Search agents")}
              inputContainerClassName="w-56 max-w-full"
              className="h-7 text-sm"
              leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
              rightElement={
                choices.isRefreshing ? (
                  <Spinner className="text-muted-foreground mr-1 size-3.5" />
                ) : search !== "" ? (
                  <Button
                    variant="ghost"
                    size="icon-xs"
                    aria-label={t("Clear search")}
                    onClick={() => narrow({ search: "" })}
                  >
                    <XIcon className="size-3" />
                  </Button>
                ) : null
              }
            />
            <SegmentedControl<AgentOrigin>
              aria-label={t("Where the agent came from")}
              value={origin}
              onValueChange={(value) => narrow({ origin: value })}
              items={[
                { value: "all", label: t("All") },
                { value: "template", label: t("From templates") },
                { value: "custom", label: t("Custom") },
              ]}
            />
          </div>
        )}
      </div>

      {choices.isLoading ? (
        <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-3" aria-busy>
          {Array.from({ length: 4 }, (_, index) => (
            <Skeleton key={index} className="rounded-surface h-18" />
          ))}
        </div>
      ) : choices.isError && agents.length === 0 ? (
        <div className="border-desk-hairline rounded-surface flex items-center gap-3 border px-4 py-3">
          <p className="text-muted-foreground min-w-0 flex-1 text-sm">
            {t("The agents could not be loaded.")}
          </p>
          <Button size="sm" variant="outline" onClick={choices.refetch}>
            {t("Try again")}
          </Button>
        </div>
      ) : agents.length === 0 ? (
        <div className="border-desk-hairline rounded-surface flex items-center gap-3 border border-dashed px-4 py-3">
          <p className="text-muted-foreground min-w-0 flex-1 text-sm">
            {choices.settledSearch !== ""
              ? t("No agents match “{0}”.", choices.settledSearch)
              : t("No agents match this filter.")}
          </p>
          <Button size="sm" variant="ghost" onClick={() => narrow({ search: "", origin: "all" })}>
            {t("Clear filters")}
          </Button>
        </div>
      ) : (
        <>
          <ul
            className={cn(
              "grid gap-2 transition-opacity sm:grid-cols-2 xl:grid-cols-3",
              choices.isRefreshing && "opacity-60",
            )}
          >
            {visible.map((agent, index) => (
              <li
                key={agent.id}
                className="animate-rise"
                style={{ animationDelay: `${Math.min(index % SHOW_MORE_STEP, 8) * 30}ms` }}
              >
                <AgentCard
                  agent={agent}
                  selected={agent.id === selectedId}
                  disabled={disabled}
                  lastUsedAgo={
                    recency.lastUsedAt.has(agent.id)
                      ? formatSecondsAgo(now - (recency.lastUsedAt.get(agent.id) ?? now))
                      : null
                  }
                  onChoose={() => onChoose(agent)}
                />
              </li>
            ))}
          </ul>

          {hasMore && (
            <div className="flex items-center justify-center gap-3">
              <Button
                variant="ghost"
                size="sm"
                onClick={showMore}
                isLoading={choices.isFetchingNextPage && showing > agents.length}
                className="text-muted-foreground hover:text-foreground"
              >
                {t("Show more")}
              </Button>
              {total !== null && (
                <span className="text-muted-foreground text-xs tabular-nums">
                  {t("Showing {0} of {1}", Math.min(showing, agents.length), total)}
                </span>
              )}
            </div>
          )}
        </>
      )}
    </section>
  );
}

function AgentCard({
  agent,
  selected,
  disabled,
  lastUsedAgo,
  onChoose,
}: {
  agent: AgentChoice;
  selected: boolean;
  disabled: boolean;
  lastUsedAgo: string | null;
  onChoose: () => void;
}) {
  const t = useT();
  const hint = agent.starters[0]?.label ?? "";

  return (
    <button
      type="button"
      disabled={disabled}
      aria-pressed={selected}
      onClick={onChoose}
      className={cn(
        "group ui-focus-ring ui-press rounded-surface flex h-full w-full items-start gap-3 border px-3 py-2.5",
        "text-left transition-colors disabled:opacity-60",
        selected
          ? "bg-surface-selected border-transparent"
          : "border-desk-hairline hover:bg-surface-hover",
      )}
    >
      <AgentTile agent={agent} size="lg" />
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="flex items-center gap-2">
          <span className="truncate text-sm font-medium">{agent.name}</span>
          {lastUsedAgo && (
            <span className="text-muted-foreground ml-auto shrink-0 text-xs tabular-nums">
              {lastUsedAgo}
            </span>
          )}
        </span>
        <span className="text-muted-foreground line-clamp-2 text-xs">
          {agent.description || (hint !== "" ? t(hint) : t("No description yet."))}
        </span>
      </span>
      <ArrowRightIcon
        aria-hidden
        className={cn(
          "text-muted-foreground mt-0.5 size-3.5 shrink-0 transition-[opacity,translate]",
          selected
            ? "opacity-0"
            : "-translate-x-1 opacity-0 group-hover:translate-x-0 group-hover:opacity-100",
        )}
      />
    </button>
  );
}
