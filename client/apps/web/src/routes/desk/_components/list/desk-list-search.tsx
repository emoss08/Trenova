import { SearchLgIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect, useRef, useState, type RefObject } from "react";

const SEARCH_DEBOUNCE_MS = 250;

/**
 * The search over a list pane. It keeps what is typed to itself and hands the
 * page a settled value a quarter second after typing stops, so a keystroke
 * re-renders this box and nothing else. Escape empties it and gives the keys
 * back to the page; Enter keeps the words and does the same, so the rows they
 * found can be walked with the list's own keys. A page that keeps searches
 * passes `onSave`, and the box offers "Save" while it has words in it.
 */
export function DeskListSearch({
  inputRef,
  placeholder,
  value,
  onCommit,
  onSave,
}: {
  inputRef: RefObject<HTMLInputElement | null>;
  placeholder: string;
  /** The committed search, from the address. */
  value: string;
  onCommit: (value: string) => void;
  onSave?: (value: string) => void;
}) {
  const t = useT();
  const [text, setText] = useState(value);
  const settled = useDebounce(text, SEARCH_DEBOUNCE_MS);
  const commit = useLatestCallback(onCommit);
  const committed = useRef(value);

  const [prevValue, setPrevValue] = useState(value);
  if (prevValue !== value) {
    setPrevValue(value);
    setText((current) => (current.trim() === value.trim() ? current : value));
  }

  useEffect(() => {
    committed.current = value;
  }, [value]);

  useEffect(() => {
    if (settled.trim() !== committed.current.trim()) {
      committed.current = settled;
      commit(settled);
    }
  }, [settled, commit]);

  const commitNow = (next: string) => {
    setText(next);
    if (next.trim() !== committed.current.trim()) {
      committed.current = next;
      commit(next);
    }
  };

  return (
    <label className="flex h-7.5 items-center gap-2 rounded-md ui-container-focus-ring bg-surface-hover pr-1 pl-2.5 text-foreground-subtle focus-within:bg-field">
      <SearchLgIcon aria-hidden className="size-3.5 shrink-0" />
      <input
        ref={inputRef}
        value={text}
        onChange={(event) => setText(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Escape") {
            event.preventDefault();
            commitNow("");
            event.currentTarget.blur();
          }
          if (event.key === "Enter") {
            event.preventDefault();
            commitNow(text);
            event.currentTarget.blur();
          }
        }}
        placeholder={placeholder}
        aria-label={placeholder}
        className="min-w-0 flex-1 bg-transparent text-sm text-foreground outline-none placeholder:text-foreground-subtle"
      />
      {text === "" ? (
        <Kbd>/</Kbd>
      ) : (
        <>
          {onSave && (
            <button
              type="button"
              onClick={() => onSave(text.trim())}
              className="h-5 rounded-sm ui-focus-ring bg-surface-active px-1.5 text-xs font-medium text-foreground hover:bg-border"
            >
              {t("Save")}
            </button>
          )}
          <button
            type="button"
            aria-label={t("Clear the search")}
            onClick={() => commitNow("")}
            className="flex size-5.5 items-center justify-center rounded-sm ui-focus-ring hover:bg-surface-active"
          >
            <XCloseIcon aria-hidden className="size-3" />
          </button>
        </>
      )}
    </label>
  );
}
