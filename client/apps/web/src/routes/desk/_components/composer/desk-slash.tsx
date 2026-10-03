import {
  commandEntries,
  fillCommand,
  parseSlashCommand,
  type CommandEntry,
} from "@/components/assistant/composer-commands";
import type { Suggestion } from "@/components/assistant/suggestions";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Fragment, useState, type KeyboardEvent, type RefObject } from "react";
import { DeskIcon, type DeskIconName } from "../desk-icons";

const COMMAND_ICONS: Record<string, DeskIconName> = {
  schedule: "receipt",
  status: "truck",
  quote: "route",
  report: "compass",
  explain: "search",
};

/** A slash typed while the box was otherwise empty: what it offers and what has been filled. */
export type DeskSlash = ReturnType<typeof useDeskSlash>;

function placeCaretAtEnd(textarea: HTMLTextAreaElement | null) {
  requestAnimationFrame(() => {
    if (textarea) {
      textarea.focus();
      textarea.setSelectionRange(textarea.value.length, textarea.value.length);
    }
  });
}

/**
 * A draft that opens with a slash lists the commands and the agent's starter
 * questions, narrowed as it grows. Picking a command with slots writes the
 * command into the box and its empty slots show as a hint after the caret;
 * Enter sends the full question once every slot is filled. Esc lets the
 * slash stand as ordinary text.
 */
export function useDeskSlash({
  value,
  onChange,
  textareaRef,
  onSendText,
  suggestions,
  enabled,
}: {
  value: string;
  onChange: (value: string) => void;
  textareaRef: RefObject<HTMLTextAreaElement | null>;
  onSendText: (text: string) => void;
  suggestions: readonly Suggestion[];
  enabled: boolean;
}) {
  const [highlighted, setHighlighted] = useState(0);
  const [dismissed, setDismissed] = useState<string | null>(null);
  const [lastQuery, setLastQuery] = useState("");
  const active = enabled && value.startsWith("/") && !value.includes("\n") && dismissed !== value;
  const query = active ? value.slice(1) : "";
  const parsed = active ? parseSlashCommand(value) : null;
  const started = parsed !== null && /\s/.test(query);
  const entries = active ? commandEntries(query, suggestions) : [];

  const head = query.split(" ")[0];
  if (head !== lastQuery) {
    setLastQuery(head);
    setHighlighted(0);
  }

  const choose = (entry: CommandEntry | undefined) => {
    if (!entry) {
      return;
    }
    if (entry.kind === "question") {
      onChange("");
      onSendText(entry.prompt);
      return;
    }
    if (entry.command.slots.length === 0) {
      onChange("");
      onSendText(fillCommand(entry.command, []));
      return;
    }
    onChange(`/${entry.command.name} `);
    placeCaretAtEnd(textareaRef.current);
  };

  /** True when the key was the menu's to handle. */
  const onKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>): boolean => {
    if (!active) {
      return false;
    }
    if (parsed && started) {
      const textarea = textareaRef.current;
      const caret = textarea ? textarea.selectionStart : value.length;
      const commandEnd = parsed.command.name.length + 2;
      if (
        event.key === "Backspace" &&
        textarea &&
        textarea.selectionStart === textarea.selectionEnd &&
        caret <= commandEnd &&
        value.slice(commandEnd).trim() === ""
      ) {
        event.preventDefault();
        onChange("");
        return true;
      }
      if (event.key === "Enter" && !event.shiftKey) {
        event.preventDefault();
        if (parsed.complete) {
          onChange("");
          onSendText(fillCommand(parsed.command, parsed.args));
        }
        return true;
      }
      if (event.key === "Escape") {
        event.preventDefault();
        setDismissed(value);
        return true;
      }
      return false;
    }
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setHighlighted((index) => Math.min(entries.length - 1, index + 1));
      return true;
    }
    if (event.key === "ArrowUp") {
      event.preventDefault();
      setHighlighted((index) => Math.max(0, index - 1));
      return true;
    }
    if ((event.key === "Enter" && !event.shiftKey) || event.key === "Tab") {
      if (entries[highlighted]) {
        event.preventDefault();
        choose(entries[highlighted]);
        return true;
      }
    }
    if (event.key === "Escape") {
      event.preventDefault();
      setDismissed(value);
      return true;
    }
    return false;
  };

  return { active, query, parsed, started, entries, highlighted, setHighlighted, choose, onKeyDown };
}

function CommandGlyph({ name }: { name: string }) {
  return (
    <span className="dk-sl-ic">
      <DeskIcon name={COMMAND_ICONS[name] ?? "chat"} size={13} stroke={2} />
    </span>
  );
}

/** The list over the box while a slash is open, or the filling guide once a command is chosen. */
export function DeskSlashMenu({ slash, agentName }: { slash: DeskSlash; agentName: string }) {
  const t = useT();
  if (!slash.active) {
    return null;
  }
  const { parsed } = slash;
  if (parsed && slash.started) {
    const { command } = parsed;
    const filled = parsed.args.filter(Boolean).length;
    return (
      <div className="dk-sl dk-sl-fill" onMouseDown={(event) => event.preventDefault()}>
        <div className="dk-sl-fh">
          <CommandGlyph name={command.name} />
          <b>{"/" + command.name}</b>
          <span>{t(command.description)}</span>
        </div>
        <div className="dk-sl-slots">
          {command.slots.map((slot, index) => (
            <span
              key={slot.name}
              className={cn(
                "dk-sl-slot",
                parsed.args[index] ? "dk-done" : index === filled && "dk-cur",
              )}
            >
              {parsed.args[index] || t(slot.hint)}
            </span>
          ))}
        </div>
        <div className="dk-sl-prev">
          <em>{t("Desk will be asked")}</em>
          {fillCommand(
            command,
            parsed.args.map((arg) => arg || "…"),
          )}
        </div>
        <div className="dk-mn-ft">
          <span>
            <span className="dk-kbd">↵</span>
            {parsed.complete ? t("ask") : t("fill every slot to ask")}
          </span>
          <span>
            <span className="dk-kbd">Esc</span>
            {t("type it as plain text")}
          </span>
        </div>
      </div>
    );
  }
  const firstQuestion = slash.entries.findIndex((entry) => entry.kind === "question");
  return (
    <div className="dk-sl" role="listbox" onMouseDown={(event) => event.preventDefault()}>
      <div className="dk-sl-l">
        {slash.entries.some((entry) => entry.kind === "command") && (
          <div className="dk-mn-h">{t("Commands")}</div>
        )}
        {slash.entries.map((entry, index) => (
          <Fragment key={entry.kind + entry.label}>
            {index === firstQuestion && <div className="dk-mn-h">{t("Ask {0}", agentName)}</div>}
            <button
              type="button"
              role="option"
              aria-selected={slash.highlighted === index}
              className={cn("dk-sl-r", slash.highlighted === index && "dk-hi")}
              onMouseMove={() => slash.highlighted !== index && slash.setHighlighted(index)}
              onClick={() => slash.choose(entry)}
            >
              {entry.kind === "command" ? (
                <>
                  <CommandGlyph name={entry.command.name} />
                  <span className="dk-sl-t">
                    <b>
                      <span className="dk-sl-nm">{entry.label}</span>
                      {entry.command.slots.map((slot) => (
                        <i key={slot.name}>{t(slot.hint)}</i>
                      ))}
                    </b>
                    <em>{t(entry.description)}</em>
                  </span>
                </>
              ) : (
                <>
                  <span className="dk-sl-ic dk-q">
                    <DeskIcon name="chat" size={13} />
                  </span>
                  <span className="dk-sl-t">
                    <b>{entry.label}</b>
                  </span>
                </>
              )}
              {slash.highlighted === index && (
                <span className="dk-kbd">
                  {entry.kind === "command" && entry.command.slots.length > 0 ? "Tab" : "↵"}
                </span>
              )}
            </button>
          </Fragment>
        ))}
        {slash.entries.length === 0 && (
          <div className="dk-mn-empty">
            {t("No command called “/{0}”. Press Esc to send it as text.", slash.query)}
          </div>
        )}
      </div>
    </div>
  );
}

/**
 * The command as typed, drawn over the textarea: the command name set apart,
 * then what has been typed, then the empty slots as a ghost hint.
 */
export function DeskSlashMirror({ slash }: { slash: DeskSlash }) {
  const t = useT();
  const { parsed } = slash;
  if (!slash.active || !parsed || !slash.started) {
    return null;
  }
  const hint = parsed.command.slots
    .filter((_, index) => !parsed.args[index])
    .map((slot) => t(slot.hint))
    .join("  ");
  return (
    <div className="dk-sl-mirror" aria-hidden>
      <span className="dk-sl-cmd">{"/" + parsed.command.name}</span>
      <span className="dk-sl-typed">{" " + slash.query.slice(parsed.command.name.length + 1)}</span>
      {hint && <span className="dk-sl-ghost">{(slash.query.endsWith(" ") ? "" : " ") + hint}</span>}
    </div>
  );
}
