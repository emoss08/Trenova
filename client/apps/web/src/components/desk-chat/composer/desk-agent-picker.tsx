import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatCompactAge } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  useCallback,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
} from "react";
import { DeskAgentTile } from "../desk-agent-tile";
import { DeskIcon } from "../desk-icons";
import { useOutsideDismiss } from "../use-outside-dismiss";

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
  useEffect(() => {
    if (openSignal > 0) {
      setOpen(true);
    }
  }, [openSignal]);
  const [query, setQuery] = useState("");
  const [highlighted, setHighlighted] = useState(0);
  const rootRef = useRef<HTMLSpanElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const listId = useId();
  // Focus that was in the popover goes back to the chip when it closes.
  const close = useCallback(() => {
    if (rootRef.current?.contains(document.activeElement)) {
      buttonRef.current?.focus();
    }
    setOpen(false);
  }, []);
  useOutsideDismiss(rootRef, open, close);
  const agentsQuery = useQuery({ ...queries.assistant.myAgents(), enabled: open });
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
    if (!open) {
      return;
    }
    const timer = window.setTimeout(() => inputRef.current?.focus(), 30);
    return () => window.clearTimeout(timer);
  }, [open]);

  const toggle = () => {
    if (!open) {
      setQuery("");
      setHighlighted(0);
    }
    setOpen(!open);
  };

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
    close();
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
      <button
        key={candidate.id}
        id={`${listId}-${position}`}
        type="button"
        role="option"
        tabIndex={-1}
        aria-selected={selected}
        data-i={position}
        className={cn("dk-ap-r", highlighted === position && "dk-hi", selected && "dk-sel")}
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
      </button>
    );
  };

  return (
    <span className="dk-ap" ref={rootRef}>
      <button
        type="button"
        ref={buttonRef}
        className={cn("dk-ap-b", open && "dk-on")}
        disabled={disabled}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={t("Asking {0}. Choose another agent", agent.name)}
        onClick={toggle}
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
      </button>
      {open && (
        <div className="dk-ap-pop" onKeyDown={onKeyDown}>
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
        </div>
      )}
    </span>
  );
}
