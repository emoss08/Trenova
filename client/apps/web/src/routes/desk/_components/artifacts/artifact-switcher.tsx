import {
  ArtifactKindIcon,
  ArtifactProvenance,
  ArtifactStatusBadge,
} from "@/components/assistant/voice/artifact-chrome";
import type { AssistantArtifact } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  CheckIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ChevronsUpDownIcon,
  PanelRightCloseIcon,
  PinIcon,
  PinOffIcon,
  SearchIcon,
} from "lucide-react";
import {
  useCallback,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
  type RefObject,
} from "react";
import { filterArtifacts } from "./artifact-order";
import { artifactProvenanceNote } from "./artifact-payloads";

/** Where a keystroke in the header goes, in the list: the index it moves to, or null to leave it. */
export function switcherTarget(key: string, current: number, count: number): number | null {
  if (count === 0) {
    return null;
  }
  const at = Math.max(0, current);
  switch (key) {
    case "ArrowRight":
    case "ArrowDown":
      return (at + 1) % count;
    case "ArrowLeft":
    case "ArrowUp":
      return (at - 1 + count) % count;
    case "Home":
      return 0;
    case "End":
      return count - 1;
    default:
      return null;
  }
}

function isTyping(target: EventTarget | null): boolean {
  return (
    target instanceof HTMLElement &&
    (target.closest("[data-slot=popover-content]") !== null ||
      target.matches("input, textarea, [contenteditable=true]"))
  );
}

/**
 * The list the switcher opens: every artifact the conversation made, pinned
 * first and newest first, with a search over it. A combobox over a listbox,
 * the way a document switcher reads to a screen reader: the arrows move a
 * highlight, Enter opens it, and a click does both at once.
 */
function ArtifactList({
  artifacts,
  activeId,
  searchRef,
  onOpen,
}: {
  artifacts: readonly AssistantArtifact[];
  activeId: string;
  /** The search box, which the popover lands focus in when it opens. */
  searchRef: RefObject<HTMLInputElement | null>;
  onOpen: (id: string) => void;
}) {
  const t = useT();
  const idBase = useId();
  const listId = `${idBase}-list`;
  const optionId = (id: string) => `${idBase}-${id}`;
  const [query, setQuery] = useState("");
  const shown = useMemo(() => filterArtifacts(artifacts, query, t), [artifacts, query, t]);
  const [highlighted, setHighlighted] = useState(() =>
    Math.max(
      0,
      artifacts.findIndex((artifact) => artifact.id === activeId),
    ),
  );

  const lit = Math.min(highlighted, shown.length - 1);
  const highlightedId = shown[lit]?.id ?? null;
  const highlightedOptionId = highlightedId === null ? undefined : optionId(highlightedId);
  useEffect(() => {
    if (highlightedOptionId !== undefined) {
      document.getElementById(highlightedOptionId)?.scrollIntoView?.({ block: "nearest" });
    }
  }, [highlightedOptionId]);

  // Left and right stay with the caret; only up and down walk the list, and
  // Enter opens what is lit.
  const onKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    switch (event.key) {
      case "Enter":
        if (highlightedId !== null) {
          event.preventDefault();
          onOpen(highlightedId);
        }
        return;
      case "ArrowDown":
      case "ArrowUp": {
        const target = switcherTarget(event.key, lit, shown.length);
        if (target !== null) {
          event.preventDefault();
          setHighlighted(target);
        }
        return;
      }
      default:
    }
  };

  return (
    <div className="flex max-h-96 min-h-0 flex-col">
      <div className="border-border-subtle shrink-0 border-b p-2">
        <Input
          ref={searchRef}
          role="combobox"
          aria-label={t("Search artifacts")}
          aria-expanded
          aria-controls={listId}
          aria-autocomplete="list"
          aria-activedescendant={highlightedOptionId}
          autoComplete="off"
          placeholder={t("Search artifacts…")}
          value={query}
          onChange={(event) => {
            setQuery(event.target.value);
            setHighlighted(0);
          }}
          onKeyDown={onKeyDown}
          leftElement={<SearchIcon className="text-foreground-subtle size-3.5" />}
          className="text-sm"
        />
      </div>
      <ul
        id={listId}
        role="listbox"
        aria-label={t("Artifacts")}
        className="scrollbar-overlay flex min-h-0 flex-1 flex-col gap-px overflow-y-auto p-1"
      >
        {shown.length === 0 && (
          <li className="text-muted-foreground px-2 py-6 text-center text-xs">
            {t("No artifact matches.")}
          </li>
        )}
        {shown.map((artifact, index) => {
          const open = artifact.id === activeId;

          return (
            // The input holds focus and the listbox is driven from it, so
            // the option itself only answers the pointer.
            <li
              key={artifact.id}
              id={optionId(artifact.id)}
              role="option"
              aria-selected={open}
              data-highlighted={index === lit || undefined}
              onMouseMove={() => setHighlighted(index)}
              onClick={() => onOpen(artifact.id)}
              className={cn(
                "flex min-w-0 cursor-pointer items-center gap-2.5 rounded-md px-2 py-1.5 transition-colors",
                "data-highlighted:bg-surface-hover",
              )}
            >
              <span className="bg-sunken text-foreground-muted flex size-7 shrink-0 items-center justify-center rounded-md">
                <ArtifactKindIcon kind={artifact.kind} className="size-3.5" />
              </span>
              <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                <span className="flex min-w-0 items-center gap-1.5">
                  <span className="truncate text-sm font-medium">{artifact.title}</span>
                  {artifact.pinned && (
                    <PinIcon aria-hidden className="text-foreground-subtle size-3 shrink-0" />
                  )}
                </span>
                <ArtifactProvenance
                  kind={artifact.kind}
                  createdAt={artifact.createdAt}
                  note={artifactProvenanceNote(artifact, t)}
                />
              </span>
              <ArtifactStatusBadge status={artifact.status} />
              {open && <CheckIcon aria-hidden className="size-3.5 shrink-0" />}
            </li>
          );
        })}
      </ul>
    </div>
  );
}

/**
 * The pane's header, read as a document switcher: the open artifact's mark
 * and title, which opens the list of all of them, where it stands in the
 * list, the way to the previous and next one, the pin and the way to hide
 * the pane. The arrows walk the list while focus is anywhere in the header,
 * and every change is said once to a screen reader.
 */
export function ArtifactSwitcher({
  artifacts,
  active,
  arrived,
  onOpen,
  onPin,
  onClose,
}: {
  /** In the order they are listed: pinned first, newest first. */
  artifacts: readonly AssistantArtifact[];
  active: AssistantArtifact;
  /** True for a moment after a turn produced a new artifact, so the header can show it landing. */
  arrived: boolean;
  onOpen: (id: string) => void;
  onPin: (id: string, pinned: boolean) => void;
  onClose: () => void;
}) {
  const t = useT();
  const [listOpen, setListOpen] = useState(false);
  const searchRef = useRef<HTMLInputElement>(null);
  // The pin settles with a small spring when it is clicked, and only then:
  // opening an artifact that is already pinned is not an action to confirm.
  const [pinTouched, setPinTouched] = useState(false);
  const count = artifacts.length;
  const index = Math.max(
    0,
    artifacts.findIndex((artifact) => artifact.id === active.id),
  );
  const position = t("{0} of {1}", index + 1, count);
  const note = artifactProvenanceNote(active, t);

  const openAt = useCallback(
    (target: number) => {
      const next = artifacts[target];
      if (next) {
        onOpen(next.id);
      }
    },
    [artifacts, onOpen],
  );

  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (isTyping(event.target)) {
      return;
    }
    const target = switcherTarget(event.key, index, count);
    if (target !== null) {
      event.preventDefault();
      openAt(target);
    }
  };

  return (
    <div
      data-slot="artifact-switcher"
      data-arrived={arrived || undefined}
      onKeyDown={onKeyDown}
      className={cn(
        "border-border-subtle bg-card flex h-12 shrink-0 items-center gap-1 border-b pr-1.5 pl-2",
        "ring-brand/40 ring-inset transition-[box-shadow] duration-700 ease-settle data-arrived:ring-1",
      )}
    >
      <span className="bg-sunken text-foreground-muted flex size-7 shrink-0 items-center justify-center rounded-md">
        <ArtifactKindIcon kind={active.kind} className="size-3.5" />
      </span>

      <div className="flex min-w-0 flex-1 flex-col items-start">
        <h3 className="flex min-w-0 max-w-full items-center">
          <Popover open={listOpen} onOpenChange={setListOpen}>
            <PopoverTrigger
              render={
                <button
                  type="button"
                  aria-haspopup="listbox"
                  className={cn(
                    "ui-focus-ring hover:bg-surface-hover -ml-1 flex h-6 min-w-0 max-w-full items-center gap-1 rounded-md px-1 transition-colors",
                    listOpen && "bg-surface-hover",
                  )}
                />
              }
            >
              <span className="truncate text-sm leading-tight font-semibold">{active.title}</span>
              <ChevronsUpDownIcon aria-hidden className="text-foreground-subtle size-3 shrink-0" />
            </PopoverTrigger>
            <PopoverContent
              align="start"
              sideOffset={6}
              initialFocus={searchRef}
              className="w-80 gap-0 p-0"
            >
              <ArtifactList
                artifacts={artifacts}
                activeId={active.id}
                searchRef={searchRef}
                onOpen={(id) => {
                  setListOpen(false);
                  onOpen(id);
                }}
              />
            </PopoverContent>
          </Popover>
        </h3>
        <ArtifactProvenance kind={active.kind} createdAt={active.createdAt} note={note} />
      </div>

      <ArtifactStatusBadge status={active.status} />

      <span className="text-foreground-subtle shrink-0 px-1 text-xs tabular-nums">{position}</span>

      <div className="flex shrink-0 items-center">
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label={t("Previous artifact")}
                disabled={count < 2}
                className="text-foreground-muted hover:text-foreground"
                onClick={() => openAt((index - 1 + count) % count)}
              />
            }
          >
            <ChevronLeftIcon className="size-4" />
          </TooltipTrigger>
          <TooltipContent>{t("Previous artifact")}</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label={t("Next artifact")}
                disabled={count < 2}
                className="text-foreground-muted hover:text-foreground"
                onClick={() => openAt((index + 1) % count)}
              />
            }
          >
            <ChevronRightIcon className="size-4" />
          </TooltipTrigger>
          <TooltipContent>{t("Next artifact")}</TooltipContent>
        </Tooltip>
      </div>

      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant="ghost"
              size="icon-sm"
              aria-label={active.pinned ? t("Unpin") : t("Pin")}
              aria-pressed={active.pinned}
              className={cn(
                "shrink-0",
                active.pinned ? "text-foreground" : "text-foreground-subtle hover:text-foreground",
              )}
              onClick={() => {
                setPinTouched(true);
                onPin(active.id, !active.pinned);
              }}
            />
          }
        >
          <span
            key={active.pinned ? "pinned" : "loose"}
            className={cn("flex", pinTouched && "animate-confirm")}
          >
            {active.pinned ? <PinOffIcon className="size-3.5" /> : <PinIcon className="size-3.5" />}
          </span>
        </TooltipTrigger>
        <TooltipContent>{active.pinned ? t("Unpin") : t("Pin to the top")}</TooltipContent>
      </Tooltip>

      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant="ghost"
              size="icon-sm"
              className="text-foreground-subtle hover:text-foreground shrink-0"
              aria-label={t("Hide artifacts")}
              onClick={onClose}
            />
          }
        >
          <PanelRightCloseIcon className="size-4" />
        </TooltipTrigger>
        <TooltipContent>{t("Hide artifacts")}</TooltipContent>
      </Tooltip>

      <p role="status" aria-label={t("Open artifact")} aria-live="polite" className="sr-only">
        {t("{0}, {1}", active.title, position)}
      </p>
    </div>
  );
}
