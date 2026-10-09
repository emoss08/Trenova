import { replaceMemoryNote, type ThreadHistory } from "@/components/assistant/thread-history";
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
import { queries } from "@/lib/queries";
import type { MemoryNote } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import {
  useCallback,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
  type ReactNode,
} from "react";
import { DeskIcon } from "../desk-icons";
import { DeskMemoryWhy } from "./desk-memory-why";
import {
  joinSteps,
  memoryDay,
  memorySource,
  memoryWhy,
  scopeLabel,
  scopeWord,
  splitSteps,
} from "./memory-format";
import "./desk-memory.css";

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

/**
 * Records a memory's new state everywhere it is cached: the Memory page's
 * reads, and every conversation whose history carries it, so the card reads
 * the same when the thread is opened again.
 */
function useSettleMemory() {
  const queryClient = useQueryClient();

  return useCallback(
    (memory: DeskMemory) => {
      queryClient.setQueriesData<ThreadHistory>(
        { queryKey: queries.assistant.messages._def },
        (history) => replaceMemoryNote(history, memory.id, cardOf(memory)),
      );
      void queryClient.invalidateQueries({ queryKey: [DESK_MEMORIES_KEY] });
    },
    [queryClient],
  );
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
  const settleMemory = useSettleMemory();
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
    onSuccess: (memory) => settleMemory(memory),
    resourceName: "Memory",
  });

  if (ids.length === 0) {
    return null;
  }

  return (
    <div className={cn("dk-mrc", open && "dk-open")}>
      <Button
        variant="quiet"
        size="sm"
        className={cn(
          "gap-1.75 pl-0.75 font-normal text-dsk-muted hover:bg-dsk-hover hover:text-dsk-fg",
          open && "bg-dsk-hover text-dsk-fg",
        )}
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
      >
        <span className="dk-mem-ic">
          <DeskIcon name="memory" size={12} stroke={2} />
        </span>
        <span>{t("{0, plural, one {Used # memory} other {Used # memories}}", live)}</span>
        <span
          className={cn(
            "grid place-items-center text-dsk-faint transition-transform duration-200 ease-(--dk-settle) motion-reduce:transition-none",
            open && "rotate-90",
          )}
        >
          <DeskIcon name="chevR" size={11} stroke={2} />
        </span>
      </Button>
      {open && cards.length > 0 && (
        <div className="dk-mrc-l">
          {cards.map((card) => (
            <div key={card.id} className={cn("dk-mrc-i", gone[card.id] && "dk-gone")}>
              <div className="dk-mrc-t">{card.content}</div>
              <div className="dk-mrc-m">
                {gone[card.id] ? (
                  <>
                    <span>{t("Forgotten. Desk won't use this again.")}</span>
                    <Button
                      variant="quiet"
                      size="xs"
                      className="ml-auto text-xs text-dsk-muted hover:text-dsk-fg"
                      disabled={forget.isPending}
                      onClick={() => forget.mutate({ id: card.id, forgotten: false })}
                    >
                      {t("Undo")}
                    </Button>
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
                      <Button
                        variant="quiet"
                        size="xs"
                        className="ml-auto text-xs text-dsk-muted hover:text-dsk-fg"
                        disabled={forget.isPending}
                        onClick={() => forget.mutate({ id: card.id, forgotten: true })}
                      >
                        {t("Forget")}
                      </Button>
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

/** Inline code in a memory, as the reply's prose shows it: `list_shipments`. */
function InlineCode({ text }: { text: string }) {
  const parts = text.split(/`([^`]+)`/);
  if (parts.length === 1) {
    return text;
  }

  return parts.map((part, index) =>
    index % 2 === 1 ? (
      <code key={index}>{part}</code>
    ) : part === "" ? null : (
      <span key={index}>{part}</span>
    ),
  );
}

/** A memory's steps as a numbered list, or its words as one paragraph when they do not split. */
function StepList({ steps, className }: { steps: readonly string[]; className?: string }) {
  if (steps.length === 1) {
    return (
      <p className={cn("dk-mem-p", className)}>
        <InlineCode text={steps[0]} />
      </p>
    );
  }

  return (
    <ol className={cn("dk-mem-st", className)}>
      {steps.map((step, index) => (
        <li key={`${index}-${step.slice(0, 24)}`}>
          <span>
            <InlineCode text={step} />
          </span>
        </li>
      ))}
    </ol>
  );
}

/** The icon, the line that says what the memory is, and what can be done with it. */
function MemoryHeader({
  icon,
  title,
  actions,
}: {
  icon: "open" | "filled" | "faint";
  title: ReactNode;
  actions?: ReactNode;
}) {
  return (
    <div className="dk-mem-h">
      <span className={cn("dk-mem-gl", `dk-${icon}`)}>
        <DeskIcon name="memory" size={14} stroke={2} />
      </span>
      <span className="dk-mem-t">{title}</span>
      {actions ? <span className="dk-mem-a">{actions}</span> : null}
    </div>
  );
}

/**
 * The memory's words, scrolled past a few lines with fades at the edges that
 * have more. Editing swaps the list for a textarea set exactly like it, one
 * step per line: Enter saves, Shift+Enter adds a line, Esc cancels.
 */
function MemorySteps({
  content,
  procedure,
  editing,
  draft,
  onDraft,
  onStartEdit,
  onCommit,
  onCancel,
}: {
  content: string;
  procedure: boolean;
  editing: boolean;
  draft: string;
  onDraft: (value: string) => void;
  onStartEdit?: () => void;
  onCommit: () => void;
  onCancel: () => void;
}) {
  const t = useT();
  const area = useRef<HTMLTextAreaElement>(null);
  const steps = useMemo(
    () => (procedure ? splitSteps(content) : [content.trim()]),
    [content, procedure],
  );

  useLayoutEffect(() => {
    const element = area.current;
    if (!editing || !element) {
      return;
    }
    element.style.height = "auto";
    element.style.height = `${element.scrollHeight}px`;
  }, [editing, draft]);

  useLayoutEffect(() => {
    const element = area.current;
    if (editing && element) {
      element.focus();
      element.setSelectionRange(element.value.length, element.value.length);
    }
  }, [editing]);

  const startFromKeys = (event: KeyboardEvent<HTMLDivElement>) => {
    if (onStartEdit && (event.key === "Enter" || event.key === " ")) {
      event.preventDefault();
      onStartEdit();
    }
  };

  return (
    <ScrollArea
      className="dk-mem-sa"
      maskHeight={14}
      maskVariant="card"
    >
      {editing ? (
        <textarea
          ref={area}
          className="dk-mem-ed"
          rows={1}
          value={draft}
          maxLength={4000}
          aria-label={t("Memory")}
          onChange={(event) => onDraft(event.target.value)}
          onKeyDown={(event) => {
            event.stopPropagation();
            if (event.key === "Enter" && !event.shiftKey) {
              event.preventDefault();
              onCommit();
            }
            if (event.key === "Escape") {
              event.preventDefault();
              onCancel();
            }
          }}
        />
      ) : onStartEdit ? (
        <div
          className="dk-mem-edit"
          role="button"
          tabIndex={0}
          aria-label={t("Edit memory")}
          title={t("Click to edit")}
          onClick={onStartEdit}
          onKeyDown={startFromKeys}
        >
          <StepList steps={steps} />
        </div>
      ) : (
        <StepList steps={steps} />
      )}
    </ScrollArea>
  );
}

/** Where the memory came from, and Why, which opens what it was kept for and what it replaced. */
function MemoryNote({
  card,
  note,
  open,
  onToggle,
}: {
  card: MemoryCard;
  note: string;
  open: boolean;
  onToggle: () => void;
}) {
  const t = useT();
  const explained = memoryWhy(card, t).length > 0;

  return (
    <>
      <div className="dk-mem-n">
        <span>{note}</span>
        {explained ? (
          <>
            <span aria-hidden>·</span>
            <Button
              variant="bare"
              size="bare"
              className="flex-none gap-0.75 text-dsk-subtle transition-colors duration-150 hover:text-dsk-fg"
              aria-expanded={open}
              onClick={onToggle}
            >
              {t("Why")}
              <span
                className={cn(
                  "grid transition-transform duration-200 ease-(--dk-settle) motion-reduce:transition-none",
                  open && "rotate-90",
                )}
              >
                <DeskIcon name="chevR" size={9} stroke={2.6} />
              </span>
            </Button>
          </>
        ) : null}
      </div>
      {explained ? (
        <div className="dk-mem-ww" inert={!open}>
          <div>
            <DeskMemoryWhy
              memory={card}
              className="dk-mem-why"
              before={(content) => <StepList steps={splitSteps(content)} />}
            />
          </div>
        </div>
      ) : null}
    </>
  );
}

type AudienceOption = {
  key: string;
  label: string;
  description: string;
  value: DeskMemoryAudience;
};

function sameAudience(a: DeskMemoryAudience, b: DeskMemoryAudience): boolean {
  return a.scope === b.scope && (a.roleId ?? null) === (b.roleId ?? null);
}

/**
 * Who the memory is for, as the word in the sentence that names them. The
 * word itself opens the choice: no chevron, no pill, a dotted underline.
 */
function ScopeWord({
  word,
  options,
  audience,
  onPick,
}: {
  word: string;
  options: readonly AudienceOption[];
  audience: DeskMemoryAudience;
  onPick: (option: AudienceOption) => void;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            variant="bare"
            size="bare"
            className="inline font-medium text-dsk-fg underline decoration-dsk-b-strong decoration-dotted underline-offset-3 transition-[text-decoration-color] duration-150 hover:decoration-current data-popup-open:decoration-current motion-reduce:transition-none"
          />
        }
        aria-label={t("Visible to {0}", word)}
      >
        {word}
      </PopoverTrigger>
      <PopoverContent
        align="start"
        alignOffset={-10}
        sideOffset={8}
        className="w-62 gap-0 rounded-lg bg-dsk-raised p-1 text-dsk-fg ui-lift-float"
      >
        <div role="radiogroup" aria-label={t("Visible to")} className="flex flex-col">
          {options.map((option) => {
            const chosen = sameAudience(option.value, audience);
            return (
              <Button
                key={option.key}
                variant="bare"
                size="bare"
                role="radio"
                aria-checked={chosen}
                className="group grid grid-cols-[14px_minmax(0,1fr)] items-start gap-x-2 rounded-md px-2 py-1.75 text-left transition-colors duration-150 hover:bg-dsk-hover"
                onClick={() => {
                  onPick(option);
                  setOpen(false);
                }}
              >
                <span className="invisible mt-0.75 text-dsk-fg group-aria-checked:visible">
                  <DeskIcon name="check" size={12} stroke={2.6} />
                </span>
                <b className="text-sm font-medium text-dsk-fg">{option.label}</b>
                <em className="col-start-2 text-xs text-dsk-subtle not-italic">
                  {option.description}
                </em>
              </Button>
            );
          })}
        </div>
      </PopoverContent>
    </Popover>
  );
}

/** How the card is drawn: asking, just kept, just turned down, already saved, or set aside. */
type CardMode = "ask" | "kept" | "no" | "saved" | "off";

/**
 * The memory a reply saved, under it. Asked first, it is one sentence —
 * "Keep these steps for everyone?" — with Not now and Keep; the steps can be
 * edited where they stand and the word "everyone" chooses who they are for.
 * Kept, it folds to a line that says so, with Undo. Already saved, it reads
 * "Learned steps" with the steps under it, and Edit and Undo on hover.
 */
export function DeskMemorySaved({ card }: { card: MemoryCard }) {
  const t = useT();
  const rt = useRichT();
  const settleMemory = useSettleMemory();
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const [state, setState] = useState<MemorySaveState>(() =>
    initialSaveState(card.status, Boolean(card.replacedBy)),
  );
  const [justKept, setJustKept] = useState(false);
  const [whyOpen, setWhyOpen] = useState(false);
  const [editing, setEditing] = useState(false);
  const [text, setText] = useState(card.content);
  const [draft, setDraft] = useState(card.content);
  const [audience, setAudience] = useState<DeskMemoryAudience>(() => audienceOf(card));
  const [scope, setScope] = useState({ scope: card.scope, roleName: card.roleName ?? null });
  const [version, setVersion] = useState(card.version);
  const [savedAt, setSavedAt] = useState(card.createdAt);
  const learned = card.source === "Reflection";
  const procedure = card.kind === "Procedure";
  const asking = state === "ask";

  const settings = useQuery({
    queryKey: [DESK_MEMORIES_KEY, "settings"],
    queryFn: ({ signal }) => fetchDeskMemorySettings(signal),
    enabled: asking || editing,
  });
  const teams = (settings.data?.roles ?? []).filter((role) => role.writable);

  const settle = (memory: DeskMemory, next: MemorySaveState) => {
    setText(memory.content);
    setDraft(memory.content);
    setVersion(memory.version);
    setSavedAt(memory.createdAt);
    setAudience(audienceOf({ scope: memory.scope, roleId: memory.roleId ?? null }));
    setScope({ scope: memory.scope, roleName: memory.roleName ?? null });
    setEditing(false);
    setState(next);
    settleMemory(memory);
  };

  const save = useApiMutation({
    mutationFn: (content: string) =>
      asking
        ? confirmDeskMemory(card.id, content, audience, version)
        : reviseDeskMemory(card.id, { content, audience, version }),
    onSuccess: (memory) => {
      setJustKept(asking);
      setWhyOpen(false);
      settle(memory, "saved");
    },
    resourceName: "Memory",
  });
  const decline = useApiMutation({
    mutationFn: () => dismissDeskMemory(card.id),
    onSuccess: (memory) => {
      setWhyOpen(false);
      settle(memory, "declined");
    },
    resourceName: "Memory",
  });
  const remove = useApiMutation({
    mutationFn: (removed: boolean) => setDeskMemoryStatus(card.id, removed ? "Retired" : "Active"),
    onSuccess: (memory, removed) => {
      setJustKept(false);
      settle(memory, removed ? "removed" : "saved");
    },
    resourceName: "Memory",
  });

  const edited = () => (procedure ? joinSteps(draft.split("\n"), text) : draft.trim());
  const startEdit = () => {
    setDraft(procedure ? splitSteps(text).join("\n") : text);
    setEditing(true);
  };
  const cancelEdit = () => {
    setDraft(text);
    setEditing(false);
  };
  const commitEdit = () => {
    const content = edited();
    if (content === "") {
      return;
    }
    if (asking) {
      setText(content);
      setEditing(false);
      return;
    }
    if (!save.isPending) {
      save.mutate(content);
    }
  };
  const keep = () => {
    const content = editing ? edited() : text;
    if (content !== "" && !save.isPending) {
      save.mutate(content);
    }
  };

  const options: AudienceOption[] = [
    {
      key: "user",
      label: t("Just you"),
      description: t("Only in your conversations"),
      value: { scope: "User", roleId: null },
    },
    ...teams.map((role) => ({
      key: role.id,
      label: role.name,
      description: t("Your team"),
      value: { scope: "Role", roleId: role.id } as DeskMemoryAudience,
    })),
  ];
  // A memory kept for everyone using the agent, or for a team the person
  // cannot change, keeps its place in the choice, so nothing is ever unchosen.
  if (!options.some((option) => sameAudience(option.value, audience))) {
    options.push({
      key: "current",
      label: scopeWord(scope, t, true),
      description:
        audience.scope === "Agent"
          ? t("Anyone using this agent")
          : audience.scope === "Organization"
            ? t("Everyone in your organization")
            : scopeLabel(scope, t),
      value: audience,
    });
  }
  const pickAudience = (option: AudienceOption) => {
    setAudience(option.value);
    setScope({
      scope: option.value.scope,
      roleName: option.value.scope === "Role" ? option.label : null,
    });
  };

  const who = scopeWord(scope, t);
  const label = learned
    ? procedure
      ? t("Learned steps")
      : t("Learned from this conversation")
    : t("Saved to memory");

  const mode: CardMode =
    state === "ask"
      ? "ask"
      : state === "declined"
        ? "no"
        : state === "saved"
          ? justKept
            ? "kept"
            : "saved"
          : "off";

  const undoLink = (onClick: () => void) => (
    <Button variant="quiet" size="sm" disabled={remove.isPending} onClick={onClick}>
      {t("Undo")}
    </Button>
  );

  let title: ReactNode;
  let actions: ReactNode = null;
  switch (mode) {
    case "ask": {
      const word = (
        <ScopeWord word={who} options={options} audience={audience} onPick={pickAudience} />
      );
      const tags = { who: () => word };
      title = (
        <b>
          {learned
            ? procedure
              ? rt("Keep these steps for <who/>?", tags)
              : rt("Keep this for <who/>?", tags)
            : rt("Remember this for <who/>?", tags)}
        </b>
      );
      actions = (
        <>
          <Button
            variant="quiet"
            size="sm"
            disabled={decline.isPending || save.isPending}
            onClick={() => decline.mutate()}
          >
            {t("Not now")}
          </Button>
          <Button
            size="sm"
            disabled={save.isPending || (editing && edited() === "")}
            onClick={keep}
          >
            {t("Keep")}
          </Button>
        </>
      );
      break;
    }
    case "kept":
      title = (
        <>
          <b>{t("Kept for {0}", who)}</b>
          <span>
            ·{" "}
            {procedure
              ? t("Desk will follow these next time")
              : t("Desk will keep this in mind")}
          </span>
        </>
      );
      actions = undoLink(() => remove.mutate(true));
      break;
    case "no":
      title = <span>{t("Not kept")}</span>;
      actions = undoLink(() => setState("ask"));
      break;
    case "saved":
      title = (
        <>
          <b>{label}</b>
          <span>
            {scopeWord(scope, t, true)} · {memoryDay(savedAt, timezone, t)}
          </span>
        </>
      );
      actions = editing ? (
        <>
          <Button variant="quiet" size="sm" onClick={cancelEdit}>
            {t("Cancel")}
          </Button>
          <Button size="sm" disabled={save.isPending || edited() === ""} onClick={commitEdit}>
            {t("Save")}
          </Button>
        </>
      ) : card.editable ? (
        <>
          <Button
            variant="quiet"
            size="icon-sm"
            aria-label={t("Edit memory")}
            title={t("Edit")}
            onClick={startEdit}
          >
            <DeskIcon name="edit" size={13} stroke={2} />
          </Button>
          <Button
            variant="quiet"
            size="icon-sm"
            aria-label={t("Undo")}
            title={t("Undo")}
            disabled={remove.isPending}
            onClick={() => remove.mutate(true)}
          >
            <DeskIcon name="undo" size={13} stroke={2} />
          </Button>
        </>
      ) : null;
      break;
    default:
      if (state === "replaced") {
        title = (
          <>
            <b>{t("Replaced by a newer memory")}</b>
            {card.replacedBy ? <span>{card.replacedBy.content}</span> : null}
          </>
        );
      } else {
        title = <span>{t("Removed from memory")}</span>;
        actions = undoLink(() => remove.mutate(false));
      }
  }

  const note =
    mode === "ask" || mode === "kept" || mode === "no"
      ? card.sourceTitle
        ? t("From “{0}”", card.sourceTitle)
        : t("From this conversation")
      : card.replaces
        ? t("Replaces earlier steps")
        : t("Saved {0} from “{1}”", memoryDay(savedAt, timezone, t), memorySource(sourceOf(card), t));

  return (
    <section
      className={cn(
        "dk-mem",
        `dk-mem-${mode}`,
        whyOpen && "dk-why-on",
        editing && "dk-editing",
      )}
      aria-label={label}
    >
      <MemoryHeader
        icon={mode === "kept" || mode === "saved" ? "filled" : mode === "ask" ? "open" : "faint"}
        title={title}
        actions={actions}
      />
      {mode === "off" ? null : (
        <div className="dk-mem-bw" inert={mode === "kept" || mode === "no"}>
          <div>
            <div className="dk-mem-b">
              <MemorySteps
                content={text}
                procedure={procedure}
                editing={editing}
                draft={draft}
                onDraft={setDraft}
                onStartEdit={mode === "ask" ? startEdit : undefined}
                onCommit={commitEdit}
                onCancel={cancelEdit}
              />
              <MemoryNote
                card={card}
                note={note}
                open={whyOpen}
                onToggle={() => setWhyOpen((value) => !value)}
              />
            </div>
          </div>
        </div>
      )}
    </section>
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
