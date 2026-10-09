"use no memo";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  createDeskMemory,
  DESK_MEMORIES_KEY,
  fetchDeskMemories,
  fetchDeskMemorySettings,
  reviseDeskMemory,
  setDeskMemoryStatus,
  setMemorySavingMode,
  type AgentMemorySavingMode,
  type DeskMemory,
  type DeskMemoryAudience,
  type DeskMemorySettings,
} from "@/lib/graphql/desk-memories";
import { useInfiniteQuery, useQuery, useQueryClient } from "@tanstack/react-query";
import { useVirtualizer } from "@tanstack/react-virtual";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import {
  useDeferredValue,
  useEffect,
  useLayoutEffect,
  useMemo,
  useReducer,
  useRef,
  useState,
  type FormEvent,
} from "react";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { deskIconClass } from "@/components/desk-chat/desk-button-styles";
import { savedLine, scopeLabel, usageLine } from "@/components/desk-chat/memory/memory-format";
import { DeskMemoryWhy } from "@/components/desk-chat/memory/desk-memory-why";
import {
  ALL_MEMORIES,
  filterScope,
  initialMemoryPageState,
  memoryChips,
  memoryPageReducer,
  memoryRows,
  sameFilter,
  type MemoryFilter,
  type MemoryRow,
} from "./memory-page-state";
import "@/components/desk-chat/memory/desk-memory.css";

const PAGE_SIZE = 50;
/** A row's height before it is measured: one line of text and the meta line. */
const ROW_ESTIMATE = 82;
/**
 * Past this many rows only those near the viewport are mounted; a shorter
 * list is laid out whole, which keeps its rows in the page's own flow.
 */
const VIRTUAL_FROM = 80;

/** An audience as a select holds it: User, Organization, or Role:{id}. */
function audienceValue(audience: { scope: string; roleId?: string | null }): string {
  return audience.scope === "Role" ? `Role:${audience.roleId ?? ""}` : audience.scope;
}

function audienceOf(value: string): DeskMemoryAudience {
  if (value.startsWith("Role:")) {
    return { scope: "Role", roleId: value.slice("Role:".length) };
  }
  return { scope: value === "Organization" ? "Organization" : "User", roleId: null };
}

/** The scopes a select offers, each enabled only where the person may keep a memory. */
function AudienceOptions({
  settings,
  current,
}: {
  settings: DeskMemorySettings | undefined;
  /** The memory being shown, whose own role is offered even when it is one the person inherits. */
  current?: DeskMemory;
}) {
  const t = useT();
  const roles = settings?.roles ?? [];
  const inherited =
    current?.scope === "Role" && current.roleId && !roles.some((role) => role.id === current.roleId)
      ? [{ id: current.roleId, name: current.roleName, writable: false }]
      : [];

  return (
    <>
      <option value="User">{t("Just you")}</option>
      {[...roles, ...inherited].map((role) => (
        <option key={role.id} value={`Role:${role.id}`} disabled={!role.writable}>
          {role.name}
        </option>
      ))}
      <option value="Organization" disabled={!settings?.canShareWithOrganization}>
        {t("Organization")}
      </option>
    </>
  );
}

/**
 * The Memory page: what Desk remembers for the person, their team and the
 * organization, how new memories are saved, and every memory to edit, move,
 * pause or forget. The list is read from the server a page at a time and only
 * the rows near the viewport are mounted, so a long memory stays quick.
 */
export function DeskMemoryPage() {
  const t = useT();
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const scrollRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  const [filter, setFilter] = useState<MemoryFilter>(ALL_MEMORIES);
  const [search, setSearch] = useState("");
  const query = useDeferredValue(search.trim());
  const [add, setAdd] = useState("");
  const [addAudience, setAddAudience] = useState("User");
  const [state, dispatch] = useReducer(memoryPageReducer, initialMemoryPageState);

  const settingsQuery = useQuery({
    queryKey: [DESK_MEMORIES_KEY, "settings"],
    queryFn: ({ signal }) => fetchDeskMemorySettings(signal),
  });
  const settings = settingsQuery.data;

  const scope = useMemo(() => filterScope(filter), [filter]);
  const listQuery = useInfiniteQuery({
    queryKey: [DESK_MEMORIES_KEY, "list", scope, query],
    initialPageParam: null as string | null,
    queryFn: ({ pageParam, signal }) =>
      fetchDeskMemories({ ...scope, query }, pageParam, PAGE_SIZE, signal),
    getNextPageParam: (page) => page.next ?? undefined,
    placeholderData: (previous) => previous,
  });
  const pages = listQuery.data?.pages;
  const listed = useMemo(() => pages?.flatMap((page) => page.items) ?? [], [pages]);
  const firstPage = pages?.[0];
  const rows = useMemo(
    () => memoryRows(listed, state, filter, query),
    [filter, listed, query, state],
  );
  const chips = useMemo(
    () =>
      memoryChips(firstPage?.counts ?? [], firstPage?.all ?? 0, settings?.roles ?? [], {
        all: t("All"),
        user: t("Just you"),
        organization: t("Organization"),
      }),
    [firstPage, settings?.roles, t],
  );

  const refresh = () => queryClient.invalidateQueries({ queryKey: [DESK_MEMORIES_KEY] });

  const modeMutation = useApiMutation({
    mutationFn: (mode: AgentMemorySavingMode) => setMemorySavingMode(mode),
    onMutate: (mode) => {
      queryClient.setQueryData<DeskMemorySettings>([DESK_MEMORIES_KEY, "settings"], (current) =>
        current ? { ...current, savingMode: mode } : current,
      );
    },
    onSuccess: (saved) => queryClient.setQueryData([DESK_MEMORIES_KEY, "settings"], saved),
    onError: () => void refresh(),
    resourceName: "Memory",
  });

  const createMutation = useApiMutation({
    mutationFn: ({ content, audience }: { content: string; audience: DeskMemoryAudience }) =>
      createDeskMemory(content, audience),
    onSuccess: (created) => {
      dispatch({ type: "added", id: created.id });
      setAdd("");
      void refresh();
    },
    resourceName: "Memory",
  });

  const reviseMutation = useApiMutation({
    mutationFn: ({
      memory,
      content,
      audience,
    }: {
      memory: DeskMemory;
      content?: string;
      audience?: DeskMemoryAudience;
    }) => reviseDeskMemory(memory.id, { content, audience, version: memory.version }),
    onSuccess: (revised) => {
      dispatch({ type: "edited", id: revised.id });
      void refresh();
    },
    onError: () => void refresh(),
    resourceName: "Memory",
  });

  const statusMutation = useApiMutation({
    mutationFn: ({
      memory,
      status,
    }: {
      memory: DeskMemory;
      status: "Active" | "Paused" | "Retired";
    }) => setDeskMemoryStatus(memory.id, status),
    onSuccess: (changed, { memory }) => {
      if (changed.status === "Retired") {
        dispatch({ type: "forgot", memory: changed });
      } else if (memory.id in state.forgotten) {
        dispatch({ type: "restored", id: memory.id });
      }
      void refresh();
    },
    resourceName: "Memory",
  });

  const create = (event: FormEvent) => {
    event.preventDefault();
    const content = add.trim();
    if (content === "" || createMutation.isPending) {
      return;
    }
    createMutation.mutate({ content, audience: audienceOf(addAudience) });
  };

  // The rows are measured against the page, which is what scrolls; the list
  // starts some way down it, under the header and the tools.
  const [listOffset, setListOffset] = useState(0);
  useLayoutEffect(() => {
    if (listRef.current) {
      setListOffset(listRef.current.offsetTop);
    }
  }, [settings, firstPage]);

  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => ROW_ESTIMATE,
    getItemKey: (index) => rows[index].memory.id,
    overscan: 6,
    scrollMargin: listOffset,
    enabled: rows.length > VIRTUAL_FROM,
  });
  const virtual = rows.length > VIRTUAL_FROM;
  const items = virtual
    ? virtualizer.getVirtualItems().map((item) => ({
        index: item.index,
        key: String(item.key),
        top: item.start - virtualizer.options.scrollMargin,
      }))
    : rows.map((row, index) => ({ index, key: row.memory.id, top: null }));

  // The next page is asked for as the end of the list comes within a screen
  // of the viewport.
  const moreRef = useRef<HTMLDivElement>(null);
  const { hasNextPage, isFetchingNextPage, fetchNextPage } = listQuery;
  useEffect(() => {
    const sentinel = moreRef.current;
    if (!sentinel || !hasNextPage || typeof IntersectionObserver === "undefined") {
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting) && !isFetchingNextPage) {
          void fetchNextPage();
        }
      },
      { root: scrollRef.current, rootMargin: "0px 0px 800px 0px" },
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [fetchNextPage, hasNextPage, isFetchingNextPage]);

  const mode = settings?.savingMode ?? "Automatic";

  return (
    <div className="dk-pg" ref={scrollRef}>
      <div className="dk-pg-in dk-narrow">
        <header className="dk-pg-h">
          <span className="dk-pg-k">{t("Memory")}</span>
          <h1>{t("What Desk remembers")}</h1>
          <p>
            {t(
              "Agents use these to answer the way your team works. Edit anything that's wrong, pause what you're unsure about, or forget it.",
            )}
          </p>
        </header>

        <div className="dk-mm-set">
          <span>
            <b>{t("Saving new memories")}</b>
            <em>
              {mode === "Automatic"
                ? t("Agents save useful facts and tell you in the conversation")
                : t("Agents ask before saving anything")}
            </em>
          </span>
          <div className="dk-mm-seg" role="radiogroup" aria-label={t("Saving new memories")}>
            {(
              [
                ["Automatic", t("Automatically")],
                ["AskFirst", t("Ask me first")],
              ] as const
            ).map(([value, label]) => (
              <Button
                key={value}
                variant="bare"
                size="bare"
                role="radio"
                aria-checked={mode === value}
                className={cn(
                  "h-7 rounded-md px-2.75 text-sm font-medium transition-colors duration-150 disabled:pointer-events-auto disabled:cursor-not-allowed disabled:opacity-100",
                  mode === value
                    ? "bg-dsk-ink text-dsk-ink-fg"
                    : "text-dsk-muted hover:not-disabled:text-dsk-fg",
                )}
                disabled={!settings}
                onClick={() => mode !== value && modeMutation.mutate(value)}
              >
                {label}
              </Button>
            ))}
          </div>
        </div>

        <form className="dk-mm-add" onSubmit={create}>
          <DeskIcon name="plus" size={14} />
          <input
            value={add}
            onChange={(event) => setAdd(event.target.value)}
            placeholder={t("Teach Desk something, e.g. “Granite pays by ACH only”")}
            aria-label={t("Add a memory")}
            maxLength={4000}
          />
          <select
            value={addAudience}
            onChange={(event) => setAddAudience(event.target.value)}
            aria-label={t("Who it is for")}
          >
            <AudienceOptions settings={settings} />
          </select>
          <Button
            type="submit"
            className="rounded-lg px-3.25 disabled:opacity-35"
            disabled={add.trim() === "" || createMutation.isPending}
          >
            {t("Save")}
          </Button>
        </form>

        <div className="dk-mm-tools">
          <div className="dk-mm-f" role="tablist" aria-label={t("Show")}>
            {chips.map((chip) => (
              <Button
                key={chip.key}
                variant="bare"
                size="bare"
                role="tab"
                aria-selected={sameFilter(filter, chip.filter)}
                className={cn(
                  "h-7 gap-1.5 rounded-full px-2.5 text-sm",
                  sameFilter(filter, chip.filter)
                    ? "bg-dsk-ink text-dsk-ink-fg"
                    : "text-dsk-muted hover:text-dsk-fg",
                )}
                onClick={() => setFilter(chip.filter)}
              >
                {chip.label}
                <em>{chip.count}</em>
              </Button>
            ))}
          </div>
          <label className="dk-ax-q">
            <DeskIcon name="search" size={13} />
            <input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder={t("Search memories")}
              aria-label={t("Search memories")}
            />
          </label>
        </div>

        <div className="dk-mm-l" ref={listRef}>
          {rows.length === 0 ? (
            listQuery.isPending ? null : (
              <div className="dk-mm-empty">{t("No memories match.")}</div>
            )
          ) : (
            <div
              className={cn(virtual && "dk-mm-v")}
              style={virtual ? { height: virtualizer.getTotalSize() } : undefined}
            >
              {items.map((item) => {
                const row = rows[item.index];
                return (
                  <MemoryRowView
                    key={item.key}
                    index={item.index}
                    measure={virtual ? virtualizer.measureElement : undefined}
                    top={item.top}
                    row={row}
                    timezone={timezone}
                    settings={settings}
                    editing={state.editing?.id === row.memory.id ? state.editing.draft : null}
                    saving={reviseMutation.isPending}
                    onEdit={() => dispatch({ type: "edit", memory: row.memory })}
                    onDraft={(text) => dispatch({ type: "draft", text })}
                    onCancel={() => dispatch({ type: "cancel-edit" })}
                    onSave={(content) => reviseMutation.mutate({ memory: row.memory, content })}
                    onMove={(audience) => reviseMutation.mutate({ memory: row.memory, audience })}
                    onStatus={(status) => statusMutation.mutate({ memory: row.memory, status })}
                  />
                );
              })}
            </div>
          )}
        </div>
        <div ref={moreRef} className="dk-mm-more" aria-hidden />
      </div>
    </div>
  );
}

function MemoryRowView({
  index,
  measure,
  top,
  row,
  timezone,
  settings,
  editing,
  saving,
  onEdit,
  onDraft,
  onCancel,
  onSave,
  onMove,
  onStatus,
}: {
  index: number;
  /** Set when the list is virtual: measures the row and places it at top. */
  measure?: (element: Element | null) => void;
  top: number | null;
  row: MemoryRow;
  timezone: string;
  settings: DeskMemorySettings | undefined;
  /** The draft while the row is being edited; null otherwise. */
  editing: string | null;
  saving: boolean;
  onEdit: () => void;
  onDraft: (text: string) => void;
  onCancel: () => void;
  onSave: (content: string) => void;
  onMove: (audience: DeskMemoryAudience) => void;
  onStatus: (status: "Active" | "Paused" | "Retired") => void;
}) {
  const t = useT();
  const { memory } = row;
  const paused = memory.status === "Paused";
  const style = top === null ? undefined : { transform: `translateY(${top}px)` };

  if (row.gone) {
    return (
      <div
        ref={measure}
        data-index={index}
        style={style}
        className={cn("dk-mm-r dk-gone", index === 0 && "dk-first")}
      >
        <span>{t("Forgotten. Agents won't use this again.")}</span>
        <Button
          variant="bare"
          size="bare"
          className="text-sm font-medium text-dsk-fg"
          onClick={() => onStatus("Active")}
        >
          {t("Undo")}
        </Button>
      </div>
    );
  }

  return (
    <div
      ref={measure}
      data-index={index}
      style={style}
      className={cn(
        "dk-mm-r",
        index === 0 && "dk-first",
        paused && "dk-paused",
        row.fresh && "dk-fresh",
      )}
    >
      {editing !== null ? (
        <div className="dk-mm-ed">
          <textarea
            // oxlint-disable-next-line jsx-a11y/no-autofocus -- the row turned into this field on purpose
            autoFocus
            value={editing}
            rows={2}
            maxLength={4000}
            aria-label={t("Memory")}
            onChange={(event) => onDraft(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Escape") {
                onCancel();
              }
            }}
          />
          <div>
            <Button
              variant="quiet"
              className="rounded-lg px-3.25 text-dsk-muted hover:bg-dsk-hover hover:text-dsk-fg"
              onClick={onCancel}
            >
              {t("Cancel")}
            </Button>
            <Button
              className="rounded-lg px-3.25"
              disabled={editing.trim() === "" || saving}
              onClick={() => onSave(editing.trim())}
            >
              {t("Save")}
            </Button>
          </div>
        </div>
      ) : (
        <>
          <p>{memory.content}</p>
          <DeskMemoryWhy memory={memory} />
        </>
      )}
      <div className="dk-mm-m">
        <select
          className="dk-mm-scope"
          value={audienceValue(memory)}
          disabled={!memory.editable}
          title={memory.editable ? undefined : scopeLabel(memory, t)}
          aria-label={t("Who it is for")}
          onChange={(event) => onMove(audienceOf(event.target.value))}
        >
          <AudienceOptions settings={settings} current={memory} />
        </select>
        <span>{savedLine(memory, timezone, t)}</span>
        <span>{usageLine(memory, timezone, t)}</span>
        {memory.editable && (
          <span className="dk-mm-acts">
            <Button
              variant="quiet"
              size="icon-sm"
              className={cn(deskIconClass, "size-6.5")}
              title={t("Edit")}
              aria-label={t("Edit")}
              onClick={onEdit}
            >
              <DeskIcon name="edit" size={13} />
            </Button>
            <Button
              variant="quiet"
              size="icon-sm"
              className={cn(deskIconClass, "size-6.5")}
              title={paused ? t("Resume") : t("Pause")}
              aria-label={paused ? t("Resume") : t("Pause")}
              onClick={() => onStatus(paused ? "Active" : "Paused")}
            >
              <DeskIcon name={paused ? "play" : "pause"} size={13} />
            </Button>
            <Button
              variant="quiet"
              size="icon-sm"
              className={cn(deskIconClass, "size-6.5")}
              title={t("Forget")}
              aria-label={t("Forget")}
              onClick={() => onStatus("Retired")}
            >
              <DeskIcon name="trash" size={13} />
            </Button>
          </span>
        )}
      </div>
    </div>
  );
}
