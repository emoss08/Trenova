import { Input } from "@trenova/shared/components/ui/input";
import { useT } from "@trenova/shared/i18n/use-t";
import { useState } from "react";

const MAX_TITLE_CHARS = 200;

/**
 * The conversation's title, edited where it is read.
 *
 * It is a field that does not look like one until you reach for it, which is
 * the right register for a name: a boxed input in the top strip would make
 * the room look like a form. Enter or leaving saves, Escape puts the saved
 * title back, and an empty title falls back to the agent's name — the same
 * thing the switcher shows for it, so the conversation never has two names.
 */
export function DeskTitleField({
  title,
  placeholder,
  onCommit,
}: {
  title: string;
  placeholder: string;
  onCommit: (title: string) => void;
}) {
  const t = useT();
  const [value, setValue] = useState(title);
  // A title saved elsewhere replaces what is typed here, derived during
  // render rather than a render later.
  const [seen, setSeen] = useState(title);
  if (seen !== title) {
    setSeen(title);
    setValue(title);
  }

  const commit = () => {
    const next = value.trim().slice(0, MAX_TITLE_CHARS);
    if (next !== title) {
      onCommit(next);
    }
  };

  return (
    <Input
      inputContainerClassName="min-w-0 w-full max-w-160"
      value={value}
      onChange={(event) => setValue(event.target.value)}
      onBlur={commit}
      onKeyDown={(event) => {
        if (event.key === "Enter") {
          event.preventDefault();
          event.currentTarget.blur();
        } else if (event.key === "Escape") {
          setValue(title);
          event.currentTarget.blur();
        }
      }}
      placeholder={placeholder}
      aria-label={t("Conversation title")}
      className="hover:bg-surface-hover focus-visible:bg-field h-8 border-transparent bg-transparent text-sm font-medium shadow-none"
    />
  );
}
