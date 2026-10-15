import { Input } from "@trenova/shared/components/ui/input";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Popover, PopoverContent } from "@trenova/shared/components/ui/popover";
import { SearchLgIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useHotkey } from "@tanstack/react-hotkeys";
import { useId, useMemo, useRef, useState, type KeyboardEvent } from "react";
import {
  buildSettingsIndex,
  searchSettings,
  SETTINGS_SECTION_META,
  type SettingsSearchResult,
} from "./settings-index";

export function SettingsSearch({
  onJump,
}: {
  onJump: (result: Pick<SettingsSearchResult, "section" | "row">) => void;
}) {
  const t = useT();
  const listId = useId();
  const anchorRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const [query, setQuery] = useState("");
  const [active, setActive] = useState(0);
  const [dismissed, setDismissed] = useState(false);
  const entries = useMemo(() => buildSettingsIndex(), []);
  const results = useMemo(() => searchSettings(entries, query, t), [entries, query, t]);
  const open = query.trim() !== "" && !dismissed;

  useHotkey(
    "Mod+K",
    () => {
      inputRef.current?.focus();
      inputRef.current?.select();
    },
    { preventDefault: true, ignoreInputs: false },
  );

  const jump = (result: SettingsSearchResult) => {
    onJump(result);
    setQuery("");
    setActive(0);
  };

  const onKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setActive((index) => Math.min(results.length - 1, index + 1));
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setActive((index) => Math.max(0, index - 1));
    } else if (event.key === "Enter") {
      const result = results[active];
      if (result) {
        event.preventDefault();
        jump(result);
      }
    } else if (event.key === "Escape" && query !== "") {
      event.preventDefault();
      event.stopPropagation();
      setQuery("");
    }
  };

  const sectionLabel = (section: string) =>
    t(SETTINGS_SECTION_META.find((meta) => meta.id === section)?.label ?? section);

  return (
    <div ref={anchorRef} className="mb-3">
      <Input
        ref={inputRef}
        type="search"
        role="combobox"
        aria-expanded={open}
        aria-controls={listId}
        aria-autocomplete="list"
        aria-activedescendant={open && results[active] ? `${listId}-${active}` : undefined}
        aria-label={t("Find a setting")}
        placeholder={t("Find a setting")}
        value={query}
        onChange={(event) => {
          setQuery(event.target.value);
          setActive(0);
          setDismissed(false);
        }}
        onKeyDown={onKeyDown}
        leftElement={<SearchLgIcon className="size-3.5" />}
        rightElement={<Kbd className="mr-1">⌘K</Kbd>}
      />
      <Popover open={open} onOpenChange={(next) => !next && setDismissed(true)}>
        <PopoverContent
          anchor={anchorRef}
          align="start"
          side="bottom"
          initialFocus={false}
          finalFocus={false}
          positionerClassName="w-(--anchor-width)"
          className="max-h-85 gap-0 overflow-y-auto p-1"
        >
          <div id={listId} role="listbox" aria-label={t("Settings")}>
            {results.length === 0 ? (
              <p className="text-foreground-subtle px-2.5 py-2.5 text-xs">
                {t("Nothing matches “{0}”", query.trim())}
              </p>
            ) : (
              results.map((result, index) => (
                <button
                  key={`${result.section}-${result.row}-${result.label}`}
                  id={`${listId}-${index}`}
                  type="button"
                  role="option"
                  aria-selected={index === active}
                  onMouseEnter={() => setActive(index)}
                  onMouseDown={(event) => event.preventDefault()}
                  onClick={() => jump(result)}
                  className={cn(
                    "flex w-full flex-col items-start gap-px rounded-md px-2.5 py-1.5 text-left",
                    index === active && "bg-surface-hover",
                  )}
                >
                  <span className="text-xs font-medium">{result.text}</span>
                  <span className="text-foreground-subtle text-2xs">
                    {result.detailText
                      ? t("{0} · {1}", sectionLabel(result.section), result.detailText)
                      : sectionLabel(result.section)}
                  </span>
                </button>
              ))
            )}
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
}
