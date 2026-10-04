import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  confirmDeskMemory,
  DESK_MEMORIES_KEY,
  dismissDeskMemory,
  fetchDeskMemoriesByIds,
  fetchDeskMemorySettings,
  reviseDeskMemory,
  setDeskMemoryStatus,
  type DeskMemory,
  type DeskMemoryAudience,
} from "@/lib/graphql/desk-memories";
import type { MemoryNote } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useLayoutEffect, useMemo, useRef, useState } from "react";
import { DeskIcon } from "../desk-icons";
import { DeskMemoryWhy } from "./desk-memory-why";
import { memoryDay, memorySource, scopeLabel } from "./memory-format";
import "../../_styles/desk-memory.css";

/** A memory as the conversation shows it: served with the reply, or read once a live turn names it. */
export type MemoryCard = Pick<
  MemoryNote,
  | "id"
  | "content"
  | "kind"
  | "scope"
  | "roleId"
  | "roleName"
  | "status"
  | "source"
  | "sourceTitle"
  | "createdAt"
  | "version"
  | "editable"
  | "reason"
  | "replaces"
  | "replacedBy"
>;

function cardOf(memory: DeskMemory): MemoryCard {
  return {
    id: memory.id,
    content: memory.content,
    kind: memory.kind,
    scope: memory.scope,
    roleId: memory.roleId ?? null,
    roleName: memory.roleName,
    status: memory.status,
    source: memory.source,
    sourceTitle: memory.sourceTitle,
    createdAt: memory.createdAt,
    version: memory.version,
    editable: memory.editable,
    reason: memory.reason,
    replaces: memory.replaces ?? null,
    replacedBy: memory.replacedBy ?? null,
  };
}

/**
 * The memories named by id, as cards: those the reply was served with, and
 * any others read once, together, when they are needed. A memory the person
 * can no longer see is left out.
 */
export function useMemoryCards(
  ids: readonly string[],
  served: readonly MemoryNote[] | null | undefined,
  enabled = true,
): MemoryCard[] {
  const known = useMemo(() => new Map((served ?? []).map((note) => [note.id, note])), [served]);
  const missing = useMemo(() => ids.filter((id) => !known.has(id)), [ids, known]);
  const read = useQuery({
    queryKey: [DESK_MEMORIES_KEY, "by-ids", missing],
    queryFn: ({ signal }) => fetchDeskMemoriesByIds(missing, signal),
    enabled: enabled && missing.length > 0,
    staleTime: 30_000,
  });

  return useMemo(() => {
    const fetched = new Map((read.data ?? []).map((memory) => [memory.id, cardOf(memory)]));
    return ids.flatMap((id) => {
      const card = known.get(id) ?? fetched.get(id);
      return card ? [card] : [];
    });
  }, [ids, known, read.data]);
}

function sourceOf(card: MemoryCard) {
  return { source: card.source, sourceTitle: card.sourceTitle ?? "", scope: card.scope };
}

/**
 * "Used 2 memories" above a reply: the memories its turn read, opened on
 * click, each with where it came from and a Forget that Undo takes back.
 */
export function DeskMemoryRecall({
  ids,
  served,
}: {
  ids: readonly string[];
  served?: readonly MemoryNote[] | null;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const [open, setOpen] = useState(false);
  const [gone, setGone] = useState<Record<string, boolean>>({});
  const cards = useMemoryCards(ids, served, open);
  const live = ids.filter((id) => !gone[id]).length;

  const forget = useApiMutation({
    mutationFn: ({ id, forgotten }: { id: string; forgotten: boolean }) =>
      setDeskMemoryStatus(id, forgotten ? "Retired" : "Active"),
    onMutate: ({ id, forgotten }) => setGone((current) => ({ ...current, [id]: forgotten })),
    onError: (_error, { id, forgotten }) =>
      setGone((current) => ({ ...current, [id]: !forgotten })),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: [DESK_MEMORIES_KEY] }),
    resourceName: "Memory",
  });

  if (ids.length === 0) {
    return null;
  }

  return (
    <div className={cn("dk-mrc", open && "dk-open")}>
      <button
        type="button"
        className="dk-mrc-b"
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
      >
        <span className="dk-mem-ic">
          <DeskIcon name="memory" size={12} stroke={2} />
        </span>
        <span>{t("{0, plural, one {Used # memory} other {Used # memories}}", live)}</span>
        <span className="dk-mrc-cv">
          <DeskIcon name="chevR" size={11} stroke={2} />
        </span>
      </button>
      {open && cards.length > 0 && (
        <div className="dk-mrc-l">
          {cards.map((card) => (
            <div key={card.id} className={cn("dk-mrc-i", gone[card.id] && "dk-gone")}>
              <div className="dk-mrc-t">{card.content}</div>
              <div className="dk-mrc-m">
                {gone[card.id] ? (
                  <>
                    <span>{t("Forgotten. Desk won't use this again.")}</span>
                    <button
                      type="button"
                      disabled={forget.isPending}
                      onClick={() => forget.mutate({ id: card.id, forgotten: false })}
                    >
                      {t("Undo")}
                    </button>
                  </>
                ) : (
                  <>
                    <span className="dk-mem-scope">{scopeLabel(card, t)}</span>
                    <span>
                      {t(
                        "Saved {0} from “{1}”",
                        memoryDay(card.createdAt, timezone, t),
                        memorySource(sourceOf(card), t),
                      )}
                    </span>
                    {card.editable && (
                      <button
                        type="button"
                        disabled={forget.isPending}
                        onClick={() => forget.mutate({ id: card.id, forgotten: true })}
                      >
                        {t("Forget")}
                      </button>
                    )}
                  </>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

/** Where a saved memory card stands: kept, offered, being edited, or taken back. */
export type MemorySaveState = "saved" | "ask" | "edit" | "removed" | "declined" | "replaced";

/**
 * A retired memory that a newer one replaced reads as replaced, not removed:
 * bringing it back would set it beside the memory that took its place.
 */
export function initialSaveState(status: string, replaced = false): MemorySaveState {
  switch (status) {
    case "Suggested":
      return "ask";
    case "Dismissed":
      return "declined";
    case "Retired":
      return replaced ? "replaced" : "removed";
    default:
      return "saved";
  }
}

function audienceOf(card: Pick<MemoryCard, "scope" | "roleId">): DeskMemoryAudience {
  if (card.scope === "Role") {
    return { scope: "Role", roleId: card.roleId ?? null };
  }
  if (card.scope === "Agent") {
    return { scope: "Agent", roleId: null };
  }
  return { scope: card.scope === "Organization" ? "Organization" : "User", roleId: null };
}

/**
 * The memory a reply saved, under it: "Saved to memory" with Edit and Undo,
 * or, for a person who asked to be asked first, "Remember this for next
 * time?" with the words to edit and who it is for, until they save it or
 * turn it down.
 */
export function DeskMemorySaved({ card }: { card: MemoryCard }) {
  const t = useT();
  const queryClient = useQueryClient();
  const [state, setState] = useState<MemorySaveState>(() =>
    initialSaveState(card.status, Boolean(card.replacedBy)),
  );
  const [text, setText] = useState(card.content);
  const [draft, setDraft] = useState(card.content);
  const [audience, setAudience] = useState<DeskMemoryAudience>(() => audienceOf(card));
  const [scopeName, setScopeName] = useState(() => scopeLabel(card, t));
  const [version, setVersion] = useState(card.version);
  const area = useRef<HTMLTextAreaElement>(null);
  const learned = card.source === "Reflection";
  const procedure = card.kind === "Procedure";

  const settings = useQuery({
    queryKey: [DESK_MEMORIES_KEY, "settings"],
    queryFn: ({ signal }) => fetchDeskMemorySettings(signal),
    enabled: state === "ask" || state === "edit",
  });
  const teams = (settings.data?.roles ?? []).filter((role) => role.writable);

  useLayoutEffect(() => {
    const element = area.current;
    if ((state === "edit" || state === "ask") && element) {
      element.style.height = "auto";
      element.style.height = `${element.scrollHeight}px`;
    }
  }, [state, draft]);

  const settle = (memory: DeskMemory, next: MemorySaveState) => {
    setText(memory.content);
    setDraft(memory.content);
    setVersion(memory.version);
    setAudience(audienceOf({ scope: memory.scope, roleId: memory.roleId ?? null }));
    setScopeName(scopeLabel(memory, t));
    setState(next);
    void queryClient.invalidateQueries({ queryKey: [DESK_MEMORIES_KEY] });
  };

  const save = useApiMutation({
    mutationFn: async (content: string) =>
      state === "ask"
        ? confirmDeskMemory(card.id, content, audience, version)
        : reviseDeskMemory(card.id, { content, audience, version }),
    onSuccess: (memory) => settle(memory, "saved"),
    resourceName: "Memory",
  });
  const decline = useApiMutation({
    mutationFn: () => dismissDeskMemory(card.id),
    onSuccess: (memory) => settle(memory, "declined"),
    resourceName: "Memory",
  });
  const remove = useApiMutation({
    mutationFn: (removed: boolean) => setDeskMemoryStatus(card.id, removed ? "Retired" : "Active"),
    onSuccess: (memory, removed) => settle(memory, removed ? "removed" : "saved"),
    resourceName: "Memory",
  });

  const commit = () => {
    const content = draft.trim();
    if (content !== "" && !save.isPending) {
      save.mutate(content);
    }
  };
  const cancelEdit = () => {
    setDraft(text);
    setAudience(audienceOf(card));
    setState("saved");
  };

  if (state === "replaced") {
    return (
      <div className="dk-mem dk-mem-off">
        <span className="dk-mem-ic">
          <DeskIcon name="memory" size={12} stroke={2} />
        </span>
        <span className="dk-mem-tx">
          <span>{t("Replaced by a newer memory")}</span>
          {card.replacedBy ? <span>“{card.replacedBy.content}”</span> : null}
        </span>
      </div>
    );
  }

  if (state === "removed" || state === "declined") {
    return (
      <div className="dk-mem dk-mem-off">
        <span className="dk-mem-ic">
          <DeskIcon name="memory" size={12} stroke={2} />
        </span>
        <span>{state === "removed" ? t("Removed from memory") : t("Not saved")}</span>
        <button
          type="button"
          className="dk-mem-l"
          disabled={remove.isPending}
          onClick={() => (state === "removed" ? remove.mutate(false) : setState("ask"))}
        >
          {t("Undo")}
        </button>
      </div>
    );
  }

  if (state === "ask" || state === "edit") {
    const asking = state === "ask";
    const chosen = (value: DeskMemoryAudience) =>
      audience.scope === value.scope && (audience.roleId ?? null) === (value.roleId ?? null);
    const options: { key: string; label: string; value: DeskMemoryAudience }[] = [
      { key: "user", label: t("Just you"), value: { scope: "User", roleId: null } },
      ...teams.map((role) => ({
        key: role.id,
        label: role.name,
        value: { scope: "Role", roleId: role.id } as DeskMemoryAudience,
      })),
    ];
    // A memory kept for a team the person cannot change keeps its place in
    // the choice, so the card never shows nothing chosen.
    if (!options.some((option) => chosen(option.value))) {
      options.push({ key: "current", label: scopeName, value: audience });
    }

    return (
      <div className={cn("dk-mem dk-mem-card", asking && "dk-ask")}>
        <div className="dk-mem-h">
          <span className="dk-mem-ic">
            <DeskIcon name="memory" size={12} stroke={2} />
          </span>
          <b>
            {asking
              ? learned
                ? procedure
                  ? t("Learned the steps that worked. Keep them for next time?")
                  : t("Learned something from this conversation. Keep it?")
                : t("Remember this for next time?")
              : t("Edit memory")}
          </b>
        </div>
        {asking ? <DeskMemoryWhy memory={card} /> : null}
        <textarea
          ref={area}
          rows={1}
          value={draft}
          maxLength={4000}
          aria-label={t("Memory")}
          onChange={(event) => setDraft(event.target.value)}
          onKeyDown={(event) => {
            event.stopPropagation();
            if (event.key === "Enter" && !event.shiftKey) {
              event.preventDefault();
              commit();
            }
            if (event.key === "Escape" && !asking) {
              cancelEdit();
            }
          }}
        />
        <div className="dk-mem-f">
          <span className="dk-mem-fl">{t("Visible to")}</span>
          <span className="dk-mem-seg" role="radiogroup" aria-label={t("Visible to")}>
            {options.map((option) => (
              <button
                key={option.key}
                type="button"
                role="radio"
                aria-checked={chosen(option.value)}
                className={cn(chosen(option.value) && "dk-on")}
                onClick={() => {
                  setAudience(option.value);
                  setScopeName(option.label);
                }}
              >
                {option.label}
              </button>
            ))}
          </span>
          <span className="dk-mem-sp" />
          <button
            type="button"
            className="dk-mem-bt"
            disabled={decline.isPending}
            onClick={() => (asking ? decline.mutate() : cancelEdit())}
          >
            {asking ? t("Don't save") : t("Cancel")}
          </button>
          <button
            type="button"
            className="dk-mem-save"
            disabled={draft.trim() === "" || save.isPending}
            onClick={commit}
          >
            {asking ? t("Save memory") : t("Save")}
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="dk-mem dk-mem-done">
      <span className="dk-mem-ic">
        <DeskIcon name="memory" size={12} stroke={2} />
      </span>
      <span className="dk-mem-tx">
        <em>
          {learned
            ? procedure
              ? t("Learned the steps that worked")
              : t("Learned from this conversation")
            : t("Saved to memory")}
        </em>
        {text}
        <DeskMemoryWhy memory={card} />
      </span>
      <span className="dk-mem-scope">{scopeName}</span>
      {card.editable && (
        <span className="dk-mem-acts">
          <button
            type="button"
            className="dk-mem-l"
            onClick={() => {
              setDraft(text);
              setState("edit");
            }}
          >
            {t("Edit")}
          </button>
          <button
            type="button"
            className="dk-mem-l"
            disabled={remove.isPending}
            onClick={() => remove.mutate(true)}
          >
            {t("Undo")}
          </button>
        </span>
      )}
    </div>
  );
}

/** The cards for what a reply saved, read together. */
export function DeskMemorySavedList({
  ids,
  served,
}: {
  ids: readonly string[];
  served?: readonly MemoryNote[] | null;
}) {
  const cards = useMemoryCards(ids, served);

  return (
    <>
      {cards.map((card) => (
        <DeskMemorySaved key={card.id} card={card} />
      ))}
    </>
  );
}
