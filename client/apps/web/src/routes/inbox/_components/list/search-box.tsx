import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { SearchLgIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect, useRef, useState, type RefObject } from "react";

const SEARCH_DEBOUNCE_MS = 250;

/**
 * The folder's search. It keeps what is typed to itself and hands the
 * folder a settled value a quarter second after typing stops, so a keystroke
 * re-renders this box and nothing else. With words in it, "Save" keeps them
 * as a view.
 */
export function SearchBox({
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
  onSave: (value: string) => void;
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

  const clear = () => {
    setText("");
    committed.current = "";
    commit("");
  };

  return (
    <label className="ui-container-focus-ring bg-surface-hover focus-within:bg-field text-foreground-subtle flex h-7.5 items-center gap-2 rounded-md pr-1 pl-2.5">
      <SearchLgIcon aria-hidden className="size-3.5 shrink-0" />
      <input
        ref={inputRef}
        value={text}
        onChange={(event) => setText(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Escape") {
            event.preventDefault();
            clear();
            event.currentTarget.blur();
          }
        }}
        placeholder={placeholder}
        aria-label={placeholder}
        className="text-foreground placeholder:text-foreground-subtle min-w-0 flex-1 bg-transparent text-sm outline-none"
      />
      {text === "" ? (
        <Kbd>/</Kbd>
      ) : (
        <>
          <button
            type="button"
            onClick={() => onSave(text.trim())}
            className="ui-focus-ring bg-surface-active text-foreground hover:bg-border h-5 rounded-sm px-1.5 text-xs font-medium"
          >
            {t("Save")}
          </button>
          <button
            type="button"
            aria-label={t("Clear the search")}
            onClick={clear}
            className="ui-focus-ring hover:bg-surface-active flex size-5.5 items-center justify-center rounded-sm"
          >
            <XCloseIcon aria-hidden className="size-3" />
          </button>
        </>
      )}
    </label>
  );
}
