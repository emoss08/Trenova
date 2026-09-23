import { AgentTile } from "@/components/agent-identity/agent-tile";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useVirtualizer } from "@tanstack/react-virtual";
import { CheckIcon, ChevronsUpDownIcon, SearchIcon } from "lucide-react";
import {
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
  type ReactElement,
} from "react";
import { useAgentChoices } from "./use-agent-choices";

/** Rows from the end at which the next page is asked for. */
const PREFETCH_ROWS = 6;
const ROW_HEIGHT = { heading: 28, agent: 44, status: 40 } as const;

type PickerRow =
  | { kind: "heading"; key: string; label: string }
  | { kind: "agent"; key: string; agent: AgentChoice; recent: boolean }
  | { kind: "status"; key: string };

export type AgentPickerProps = {
  /** The agent questions go to now. */
  agent: AgentChoice | null;
  onSelect: (agent: AgentChoice) => void;
  /** The person's agents, most recent first, listed ahead of the rest. */
  recentIds: readonly string[];
  lastUsedAt?: ReadonlyMap<string, number>;
  disabled?: boolean;
  align?: "start" | "center" | "end";
  side?: "top" | "bottom";
  /**
   * The control that opens the picker, when the default chip is not the
   * right shape. It is given the open state through the trigger.
   */
  trigger?: ReactElement;
  className?: string;
};

const nowInSeconds = () => Math.floor(Date.now() / 1000);

/**
 * Who a question goes to.
 *
 * One control, showing the agent, that opens onto a search and a list. It
 * replaces a row of chips — one per agent — that was fine at six agents and
 * a wall at sixty. The list is read from the server a page at a time and
 * only the rows in view are mounted, so an organization with hundreds of
 * agents opens it as quickly as one with three. The agents the person has
 * asked before lead the list, because most questions go to the same two.
 */
export function AgentPicker({
  agent,
  onSelect,
  recentIds,
  lastUsedAt,
  disabled = false,
  align = "start",
  side = "bottom",
  trigger,
  className,
}: AgentPickerProps) {
  const t = useT();
  const [open, setOpen] = useState(false);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        disabled={disabled}
        render={
          trigger ?? (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              aria-label={
                agent ? t("Asking {0}. Choose another agent", agent.name) : t("Choose an agent")
              }
              className={cn(
                "text-foreground data-popup-open:bg-surface-active h-7 max-w-60 gap-1.5 rounded-full pr-2 pl-1",
                className,
              )}
            />
          )
        }
      >
        {trigger ? undefined : (
          <>
            {/* Re-keyed on the agent, so a new choice lands with a small
                confirming settle rather than a silent swap. */}
            <AgentTile
              key={agent?.id ?? "none"}
              agent={agent}
              size="xs"
              className="animate-confirm"
            />
            <span className="min-w-0 truncate text-sm">{agent?.name ?? t("Choose an agent")}</span>
            <ChevronsUpDownIcon className="text-muted-foreground size-3.5" />
          </>
        )}
      </PopoverTrigger>
      <PopoverContent
        align={align}
        side={side}
        sideOffset={6}
        className="ui-lift-float w-80 gap-0 overflow-hidden p-0"
      >
        {open && (
          <AgentPickerList
            selectedId={agent?.id ?? null}
            recentIds={recentIds}
            lastUsedAt={lastUsedAt}
            onSelect={(picked) => {
              onSelect(picked);
              setOpen(false);
            }}
          />
        )}
      </PopoverContent>
    </Popover>
  );
}

const NO_HIDDEN: ReadonlySet<string> = new Set();

export type AgentPickerListProps = {
  selectedId: string | null;
  recentIds: readonly string[];
  lastUsedAt?: ReadonlyMap<string, number>;
  onSelect: (agent: AgentChoice) => void;
  /**
   * Agents left out of the list: the one being configured, and those already
   * chosen where several are picked one at a time.
   */
  hiddenIds?: ReadonlySet<string>;
  /** Said when there is nothing to pick before any search narrows the list. */
  emptyMessage?: string;
};

/**
 * The searchable, paged list of agents a person can ask, without the
 * control that opens it. The composer's picker opens it to choose who a
 * question goes to; AI Control opens it to choose who an agent may ask.
 */
export function AgentPickerList({
  selectedId,
  recentIds,
  lastUsedAt,
  onSelect,
  hiddenIds = NO_HIDDEN,
  emptyMessage,
}: AgentPickerListProps) {
  const t = useT();
  const listId = useId();
  const [now] = useState(nowInSeconds);
  const [search, setSearch] = useState("");
  const [active, setActive] = useState(0);
  const scrollRef = useRef<HTMLDivElement>(null);
  const choices = useAgentChoices({ search, origin: "all", recentIds });

  const rows = useMemo<PickerRow[]>(() => {
    const recent = choices.recent.filter((agent) => !hiddenIds.has(agent.id));
    const items = choices.items.filter((agent) => !hiddenIds.has(agent.id));
    const next: PickerRow[] = [];
    if (recent.length > 0) {
      next.push({ kind: "heading", key: "h-recent", label: t("Recent") });
      for (const recentAgent of recent) {
        next.push({ kind: "agent", key: `r-${recentAgent.id}`, agent: recentAgent, recent: true });
      }
      if (items.length > 0) {
        next.push({ kind: "heading", key: "h-all", label: t("All agents") });
      }
    }
    for (const item of items) {
      next.push({ kind: "agent", key: `a-${item.id}`, agent: item, recent: false });
    }
    if (choices.hasNextPage) {
      next.push({ kind: "status", key: "status" });
    }

    return next;
  }, [choices.hasNextPage, choices.items, choices.recent, hiddenIds, t]);

  const agentIndexes = useMemo(
    () => rows.flatMap((row, index) => (row.kind === "agent" ? [index] : [])),
    [rows],
  );
  const activeIndex = agentIndexes[Math.min(active, agentIndexes.length - 1)] ?? -1;

  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: (index) => ROW_HEIGHT[rows[index].kind],
    getItemKey: (index) => rows[index].key,
    overscan: 8,
    initialRect: { width: 320, height: 320 },
  });
  const virtualItems = virtualizer.getVirtualItems();
  const lastVisible = virtualItems.at(-1)?.index ?? 0;

  const { fetchNextPage, hasNextPage, isFetchingNextPage } = choices;
  useEffect(() => {
    if (hasNextPage && !isFetchingNextPage && lastVisible >= rows.length - PREFETCH_ROWS) {
      fetchNextPage();
    }
  }, [fetchNextPage, hasNextPage, isFetchingNextPage, lastVisible, rows.length]);

  const moveTo = (position: number) => {
    if (agentIndexes.length === 0) {
      return;
    }
    const bounded = Math.max(0, Math.min(position, agentIndexes.length - 1));
    setActive(bounded);
    virtualizer.scrollToIndex(agentIndexes[bounded], { align: "auto" });
  };

  const onKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    const position = Math.min(active, agentIndexes.length - 1);
    switch (event.key) {
      case "ArrowDown":
        event.preventDefault();
        moveTo(position + 1);
        break;
      case "ArrowUp":
        event.preventDefault();
        moveTo(position - 1);
        break;
      case "Home":
        event.preventDefault();
        moveTo(0);
        break;
      case "End":
        event.preventDefault();
        moveTo(agentIndexes.length - 1);
        break;
      case "Enter": {
        const row = rows[activeIndex];
        if (row?.kind === "agent") {
          event.preventDefault();
          onSelect(row.agent);
        }
        break;
      }
      default:
        break;
    }
  };

  const optionId = (index: number) => `${listId}-${index}`;
  const empty = !choices.isLoading && rows.length === 0;

  return (
    <div className="flex flex-col">
      <div className="border-border border-b p-2">
        <Input
          autoFocus
          value={search}
          onChange={(event) => {
            setSearch(event.target.value);
            setActive(0);
          }}
          onKeyDown={onKeyDown}
          placeholder={t("Search agents")}
          aria-label={t("Search agents")}
          role="combobox"
          aria-expanded
          aria-controls={listId}
          aria-activedescendant={activeIndex >= 0 ? optionId(activeIndex) : undefined}
          aria-autocomplete="list"
          className="h-8 text-sm"
          leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
          rightElement={
            choices.isRefreshing ? (
              <Spinner className="text-muted-foreground mr-1 size-3.5" />
            ) : null
          }
        />
      </div>

      {choices.isLoading ? (
        <div className="flex flex-col gap-1.5 p-2" aria-busy>
          <Skeleton className="h-9" />
          <Skeleton className="h-9" />
          <Skeleton className="h-9" />
        </div>
      ) : choices.isError && rows.length === 0 ? (
        <div className="flex flex-col items-center gap-2 px-4 py-6 text-center">
          <p className="text-muted-foreground text-sm">{t("The agents could not be loaded.")}</p>
          <Button size="xs" variant="outline" onClick={choices.refetch}>
            {t("Try again")}
          </Button>
        </div>
      ) : empty ? (
        <p className="text-muted-foreground px-4 py-6 text-center text-sm">
          {choices.settledSearch === ""
            ? (emptyMessage ?? t("No agents are available to talk to yet."))
            : t("No agents match “{0}”.", choices.settledSearch)}
        </p>
      ) : (
        <div
          ref={scrollRef}
          id={listId}
          role="listbox"
          aria-label={t("Agents")}
          className={cn(
            "scrollbar-overlay max-h-80 overflow-y-auto overscroll-contain p-1 transition-opacity",
            choices.isRefreshing && "opacity-70",
          )}
        >
          <div className="relative w-full" style={{ height: virtualizer.getTotalSize() }}>
            {virtualItems.map((item) => {
              const row = rows[item.index];

              return (
                <div
                  key={item.key}
                  data-index={item.index}
                  ref={virtualizer.measureElement}
                  className="absolute inset-x-0 top-0"
                  style={{ transform: `translateY(${item.start}px)` }}
                >
                  {row.kind === "heading" ? (
                    <p
                      role="presentation"
                      className="text-muted-foreground px-2 pt-2 pb-1 text-xs font-medium"
                    >
                      {row.label}
                    </p>
                  ) : row.kind === "status" ? (
                    <div role="presentation" className="flex h-10 items-center justify-center">
                      <Spinner className="text-muted-foreground size-3.5" />
                    </div>
                  ) : (
                    <AgentOption
                      id={optionId(item.index)}
                      agent={row.agent}
                      selected={row.agent.id === selectedId}
                      active={item.index === activeIndex}
                      usedAgo={
                        row.recent && lastUsedAt?.has(row.agent.id)
                          ? formatSecondsAgo(now - (lastUsedAt.get(row.agent.id) ?? now))
                          : null
                      }
                      onHover={() => setActive(agentIndexes.indexOf(item.index))}
                      onSelect={() => onSelect(row.agent)}
                    />
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

function AgentOption({
  id,
  agent,
  selected,
  active,
  usedAgo,
  onHover,
  onSelect,
}: {
  id: string;
  agent: AgentChoice;
  selected: boolean;
  active: boolean;
  usedAgo: string | null;
  onHover: () => void;
  onSelect: () => void;
}) {
  return (
    <div
      id={id}
      role="option"
      aria-selected={selected}
      data-active={active || undefined}
      onMouseMove={onHover}
      onMouseDown={(event) => event.preventDefault()}
      onClick={onSelect}
      className={cn(
        "flex h-11 cursor-pointer items-center gap-2.5 rounded-md px-2 transition-colors",
        active && "bg-surface-hover",
      )}
    >
      <AgentTile agent={agent} size="sm" />
      <span className="flex min-w-0 flex-1 flex-col">
        <span className="truncate text-sm">{agent.name}</span>
        {agent.description !== "" && (
          <span className="text-muted-foreground truncate text-xs">{agent.description}</span>
        )}
      </span>
      {usedAgo && !selected && (
        <span className="text-muted-foreground shrink-0 text-xs tabular-nums">{usedAgo}</span>
      )}
      {selected && <CheckIcon className="text-brand size-3.5 shrink-0" />}
    </div>
  );
}
