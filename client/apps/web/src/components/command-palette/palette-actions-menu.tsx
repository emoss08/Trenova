import { dropdownMenuSurfaceClassName } from "@trenova/shared/components/ui/dropdown-menu";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { rankMatches } from "@trenova/shared/lib/fuzzy-score";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useMemo, useRef, useState } from "react";
import type { PaletteAction, PaletteIcon } from "./palette-model";
import { MatchText, PaletteTile, ShortcutKeys } from "./palette-parts";

export function actionFields(action: PaletteAction) {
  return [{ key: "title" as const, text: action.label }];
}

/**
 * Everything that can be done with the selected row, over the preview. It
 * takes the keyboard while it is open — its own filter, arrows and Enter —
 * and gives it back to the search box on Escape, ⌘K, or ← and Backspace on
 * an empty filter, so the list behind it never moves.
 */
export function PaletteActionsMenu({
  title,
  icon,
  initials,
  actions,
  onRun,
  onClose,
}: {
  title: string;
  icon?: PaletteIcon;
  initials?: string;
  actions: readonly PaletteAction[];
  onRun: (action: PaletteAction) => void;
  onClose: () => void;
}) {
  const t = useT();
  const [filter, setFilter] = useState("");
  const [index, setIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const listRef = useRef<HTMLDivElement | null>(null);

  const ranked = useMemo(() => rankMatches(filter, actions, actionFields), [actions, filter]);
  const active = Math.min(index, Math.max(0, ranked.length - 1));

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  useEffect(() => {
    listRef.current
      ?.querySelector<HTMLElement>(`[data-index="${active}"]`)
      ?.scrollIntoView({ block: "nearest" });
  }, [active]);

  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    const mod = event.metaKey || event.ctrlKey;
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      if (ranked.length === 0) {
        return;
      }
      const step = event.key === "ArrowDown" ? 1 : -1;
      setIndex((active + step + ranked.length) % ranked.length);
      return;
    }
    if (event.key === "Enter") {
      event.preventDefault();
      const chosen = ranked[active];
      if (chosen) {
        onRun(chosen.entry);
      }
      return;
    }
    if (
      event.key === "Escape" ||
      (mod && event.key.toLowerCase() === "k") ||
      ((event.key === "ArrowLeft" || event.key === "Backspace") && filter === "")
    ) {
      event.preventDefault();
      event.stopPropagation();
      onClose();
    }
  };

  return (
    <div
      role="dialog"
      aria-label={t("Actions for {0}", title)}
      onMouseDown={(event) => {
        if (event.target !== inputRef.current) {
          event.preventDefault();
        }
      }}
      className={cn(
        dropdownMenuSurfaceClassName,
        "animate-pop-in ui-lift-float absolute right-2 bottom-2 z-30 flex max-h-[min(420px,calc(100%-1rem))] w-75 flex-col overflow-hidden",
      )}
    >
      <div className="border-border-subtle flex h-11 shrink-0 items-center gap-2 border-b px-3">
        <PaletteTile icon={icon} initials={initials} size="chip" />
        <span className="text-foreground min-w-0 flex-1 truncate text-base font-medium">
          {title}
        </span>
        <Kbd>Esc</Kbd>
      </div>
      <input
        ref={inputRef}
        value={filter}
        onChange={(event) => {
          setFilter(event.target.value);
          setIndex(0);
        }}
        onKeyDown={handleKeyDown}
        placeholder={t("Filter actions…")}
        aria-label={t("Filter actions")}
        spellCheck={false}
        autoComplete="off"
        className="border-border-subtle text-foreground placeholder:text-foreground-subtle h-9 shrink-0 border-b bg-transparent px-3 text-base outline-hidden"
      />
      <ScrollArea className="min-h-0 flex-1" maskHeight={0}>
        <div ref={listRef} role="listbox" aria-label={t("Actions")} className="flex flex-col p-1">
          {ranked.map(({ entry: action, match }, position) => (
            <button
              key={action.id}
              type="button"
              role="option"
              aria-selected={position === active}
              data-index={position}
              tabIndex={-1}
              onMouseMove={() => setIndex(position)}
              onClick={() => onRun(action)}
              className={cn(
                "rounded-control flex h-7.5 w-full shrink-0 items-center gap-2.5 px-2 text-left text-base transition-colors",
                position === active ? "bg-surface-hover" : undefined,
                action.destructive ? "text-danger" : "text-foreground",
              )}
            >
              <action.icon className="text-foreground-subtle size-3.5 shrink-0" />
              <MatchText
                text={action.label}
                ranges={match.highlights.title}
                className="min-w-0 flex-1 truncate"
              />
              {action.shortcut && <ShortcutKeys keys={action.shortcut} />}
            </button>
          ))}
          {ranked.length === 0 && (
            <p className="text-foreground-subtle px-2 py-2.5 text-sm">{t("No matching actions")}</p>
          )}
        </div>
      </ScrollArea>
    </div>
  );
}
