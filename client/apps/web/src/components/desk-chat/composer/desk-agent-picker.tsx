import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { assistant } from "@/lib/queries/assistant";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatCompactAge } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useId, useMemo, useRef, useState, type KeyboardEvent } from "react";
import { DeskAgentTile } from "../desk-agent-tile";
import { DeskIcon } from "../desk-icons";
import { deskPopoverClass, useComposerPopover } from "../desk-popover";

const nowInSeconds = () => Math.floor(Date.now() / 1000);

/**
 * The agent a message goes to, as a pill in the composer, and the list to
 * choose another from: the person's recent agents with when they last used
 * each, then every other agent they may ask, each with its mark and a line on
 * what it does. ↑ ↓ Home End move, Enter picks, Esc closes.
 */
export function DeskAgentPicker({
  agent,
  onSelect,
  recentIds,
  lastUsedAt,
  disabled = false,
  openSignal = 0,
}: {
  agent: AgentChoice;
  onSelect: (agent: AgentChoice) => void;
  recentIds: readonly string[];
  lastUsedAt?: ReadonlyMap<string, number>;
  disabled?: boolean;
  /** Opens the list each time it changes, for "Ask another agent" elsewhere. */
  openSignal?: number;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [highlighted, setHighlighted] = useState(0);
  const onOpenChange = (next: boolean) => {
    if (next) {
      setQuery("");
      setHighlighted(0);
    }
    setOpen(next);
  };
  useEffect(() => {
    if (openSignal > 0) {
      setOpen(true);
    }
  }, [openSignal]);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const listId = useId();
  const { anchor, wide } = useComposerPopover();
  const agentsQuery = useQuery({ ...assistant.myAgents(), enabled: open });
  const [now] = useState(nowInSeconds);

  const { recent, rest } = useMemo(() => {
    const term = query.trim().toLowerCase();
    const matches = (candidate: AgentChoice) =>
      term === "" || `${candidate.name} ${candidate.description}`.toLowerCase().includes(term);
    const all = agentsQuery.data ?? [];
    const byId = new Map(all.map((candidate) => [candidate.id, candidate]));
    const recentList = recentIds
      .map((id) => byId.get(id))
      .filter(
        (candidate): candidate is AgentChoice => candidate !== undefined && matches(candidate),
      );
    const recentSet = new Set(recentList.map((candidate) => candidate.id));
    return {
      recent: recentList,
      rest: all.filter((candidate) => !recentSet.has(candidate.id) && matches(candidate)),
    };
  }, [agentsQuery.data, query, recentIds]);
  const flat = useMemo(() => [...recent, ...rest], [recent, rest]);

  useEffect(() => {
    listRef.current
      ?.querySelector<HTMLElement>(`[data-i="${highlighted}"]`)
      ?.scrollIntoView({ block: "nearest" });
  }, [highlighted]);

  const pick = (candidate: AgentChoice | undefined) => {
    if (!candidate) {
      return;
    }
    onSelect(candidate);
    setOpen(false);
  };

  const onKeyDown = (event: KeyboardEvent) => {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setHighlighted((index) => Math.max(0, Math.min(flat.length - 1, index + 1)));
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setHighlighted((index) => Math.max(0, index - 1));
    } else if (event.key === "Home") {
      event.preventDefault();
      setHighlighted(0);
    } else if (event.key === "End") {
      event.preventDefault();
      setHighlighted(Math.max(0, flat.length - 1));
    } else if (event.key === "Enter") {
      event.preventDefault();
      pick(flat[highlighted]);
    }
  };

  const row = (candidate: AgentChoice, isRecent: boolean, position: number) => {
    const selected = candidate.id === agent.id;
    const used = lastUsedAt?.get(candidate.id);
    return (
      <Button
        key={candidate.id}
        id={`${listId}-${position}`}
        variant="bare"
        size="bare"
        role="option"
        tabIndex={-1}
        aria-selected={selected}
        data-i={position}
        className={cn(
          "flex h-11 w-full animate-[dk-mprow_260ms_var(--dk-settle)_both] gap-2.5 rounded-lg px-2 text-left",
          highlighted === position && "bg-dsk-hover",
        )}
        onMouseMove={() => highlighted !== position && setHighlighted(position)}
        onClick={() => pick(candidate)}
      >
        <DeskAgentTile agent={candidate} size="sm" />
        <span className="dk-ap-t">
          <b>{candidate.name}</b>
          <span>{candidate.description}</span>
        </span>
        {isRecent && !selected && used !== undefined && (
          <span className="dk-ap-ago">{formatCompactAge(now - used)}</span>
        )}
        {selected && (
          <span className="dk-ap-ck">
            <DeskIcon name="check" size={13} stroke={2.4} />
          </span>
        )}
      </Button>
    );
  };

  return (
    <Popover open={open} onOpenChange={onOpenChange}>
      <PopoverTrigger
        disabled={disabled}
        render={
          <Button
            variant="bare"
            size="bare"
            className={cn(
              "h-7 max-w-55 gap-1.75 rounded-badge pr-1.75 pl-1 text-sm text-dsk-fg2 transition-colors duration-150 hover:bg-dsk-hover hover:text-dsk-fg",
              open && "bg-dsk-hover text-dsk-fg",
            )}
            aria-haspopup="listbox"
            aria-label={t("Asking {0}. Choose another agent", agent.name)}
          />
        }
      >
        <DeskAgentTile key={agent.id} agent={agent} size="xs" className="dk-at-confirm" />
        <span className="dk-ap-bn">{agent.name}</span>
        <svg
          width="11"
          height="11"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2.2"
          strokeLinecap="round"
          strokeLinejoin="round"
          className="dk-ap-cv"
          aria-hidden
        >
          <path d="M8 9l4-4 4 4M8 15l4 4 4-4" />
        </svg>
      </PopoverTrigger>
      <PopoverContent
        side="top"
        align="start"
        alignOffset={wide ? 0 : -6}
        sideOffset={8}
        anchor={anchor}
        initialFocus={inputRef}
        className={cn(deskPopoverClass, wide ? "w-(--anchor-width)" : "w-80")}
        onKeyDown={onKeyDown}
      >
        <div className="dk-ap-s">
          <DeskIcon name="search" size={13} />
          <input
            ref={inputRef}
            value={query}
            onChange={(event) => {
              setQuery(event.target.value);
              setHighlighted(0);
            }}
            placeholder={t("Search agents")}
            role="combobox"
            aria-expanded
            aria-autocomplete="list"
            aria-controls={listId}
            aria-activedescendant={flat[highlighted] ? `${listId}-${highlighted}` : undefined}
            aria-label={t("Search agents")}
          />
        </div>
        <ScrollArea viewportClassName="max-h-[min(320px,calc(100vh-260px))]" maskVariant="raised">
          <div
            className="dk-ap-l"
            ref={listRef}
            role="listbox"
            id={listId}
            aria-label={t("Search agents")}
          >
            {agentsQuery.isPending ? (
              <div className="dk-ap-empty">{t("Loading agents…")}</div>
            ) : (
              <>
                {recent.length > 0 && <div className="dk-ap-h">{t("Recent")}</div>}
                {recent.map((candidate, index) => row(candidate, true, index))}
                {recent.length > 0 && rest.length > 0 && (
                  <div className="dk-ap-h">{t("All agents")}</div>
                )}
                {rest.map((candidate, index) => row(candidate, false, recent.length + index))}
                {flat.length === 0 && (
                  <div className="dk-ap-empty">{t("No agents match “{0}”", query)}</div>
                )}
              </>
            )}
          </div>
        </ScrollArea>
      </PopoverContent>
    </Popover>
  );
}
