import { isTypingTarget, isWithinDialog } from "@/lib/dom";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect, useRef, type ReactNode } from "react";
import {
  DeskListHead,
  DeskListProgress,
  DeskListTabs,
  DeskListTitle,
  DeskListToggle,
  DeskListTokens,
  type DeskListToken,
} from "../list/desk-list-head";
import { DeskListSearch } from "../list/desk-list-search";
import { NEEDS_LENSES, lensLabel, type NeedsLens } from "./lens";

/**
 * What a lens says in the page's head: how far along its work is, whose
 * cases it lists, what it is searched for and what it is narrowed to. The
 * head itself is one and stays; the lens in front fills it.
 */
export type NeedsHeadLens = {
  /** How much of today's work is behind the person; null until it has been read. */
  progress: { label: string; done: number; total: number } | null;
  /** Whose cases are listed; null under a lens that lists everyone's, or for a reader who only sees their own. */
  whose: "mine" | "everyone" | null;
  onWhose?: (whose: "mine" | "everyone") => void;
  /** The words the lens is searched for. */
  search: { value: string; onCommit: (value: string) => void };
  /** The lens's filter menu, drawn at the end of the lens row. */
  filter: ReactNode;
  /** What the lens is narrowed to, one token per filter. */
  tokens: readonly DeskListToken[];
  onClearFilters: () => void;
};

/**
 * The head of Needs you's list. It is drawn once for the page and every lens
 * fills the same parts of it: the page's name, whose cases it lists, how much
 * of today's work is behind the person, the lenses with what waits in each,
 * and what the lens is searched and narrowed by. So going from one lens to
 * another moves the marker and changes the words, and nothing leaves the
 * page. The keys 1 to 3 pick a lens wherever the page is, / goes to the search
 * and F opens the filter. The page holds whether the filter is open, since
 * the lens's own keys stand down while it is.
 */
export function NeedsHead({
  className,
  lens,
  lenses,
  onLens,
  onLensIntent,
  counts,
  keysEnabled = true,
  onOpenFilter,
  progress,
  whose,
  onWhose,
  search,
  filter,
  tokens,
  onClearFilters,
}: NeedsHeadLens & {
  className?: string;
  lens: NeedsLens;
  /** The lenses the person may open, in page order. */
  lenses: readonly NeedsLens[];
  onLens: (lens: NeedsLens) => void;
  /** A lens is about to be picked: the pointer or the focus is on its tab. */
  onLensIntent?: (lens: NeedsLens) => void;
  /** What waits in each lens; a lens not counted yet shows no number. */
  counts: Readonly<Partial<Record<NeedsLens, number>>>;
  /** Off while a menu is open over the page. */
  keysEnabled?: boolean;
  /** Opens the filter menu, for the F key. */
  onOpenFilter: () => void;
}) {
  const t = useT();
  const searchRef = useRef<HTMLInputElement>(null);

  const onKey = useLatestCallback((event: KeyboardEvent) => {
    if (event.defaultPrevented || event.metaKey || event.ctrlKey || event.altKey) {
      return;
    }
    if (isTypingTarget(event.target) || isWithinDialog(event.target)) {
      return;
    }
    if (event.key === "/") {
      event.preventDefault();
      searchRef.current?.focus();
      return;
    }
    if (event.key.toLowerCase() === "f") {
      event.preventDefault();
      onOpenFilter();
      return;
    }
    const picked = NEEDS_LENSES[["1", "2", "3"].indexOf(event.key)];
    if (picked && lenses.includes(picked)) {
      event.preventDefault();
      onLens(picked);
    }
  });
  useEffect(() => {
    if (!keysEnabled) {
      return;
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [keysEnabled, onKey]);

  return (
    <DeskListHead className={className}>
      <DeskListTitle title={t("Needs you")} trailing={progress?.label}>
        {whose !== null && onWhose && (
          <DeskListToggle
            label={t("Whose cases")}
            value={whose}
            onChange={onWhose}
            options={[
              { value: "mine", label: t("Mine") },
              { value: "everyone", label: t("Everyone") },
            ]}
          />
        )}
      </DeskListTitle>
      <DeskListProgress
        label={progress?.label ?? t("Reading what needs you")}
        done={progress?.done ?? 0}
        total={progress?.total ?? 0}
      />
      <DeskListTabs
        label={t("Lens")}
        value={lens}
        onChange={onLens}
        onIntent={onLensIntent}
        tabs={lenses.map((value) => ({
          value,
          label: lensLabel(t, value),
          count: counts[value],
          hint: t("Press {0}", NEEDS_LENSES.indexOf(value) + 1),
        }))}
      >
        {filter}
      </DeskListTabs>
      <div className="px-3 pt-2">
        <DeskListSearch
          inputRef={searchRef}
          placeholder={t("Search what needs you")}
          value={search.value}
          onCommit={search.onCommit}
        />
      </div>
      <DeskListTokens tokens={tokens} onClear={onClearFilters} />
    </DeskListHead>
  );
}
