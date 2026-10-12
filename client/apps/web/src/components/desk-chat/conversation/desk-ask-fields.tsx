import type { AskField } from "@/components/assistant/ask-requests";
import { askFieldsAnswer, askInputAnswer, type AskAnswer, type AskValue } from "@/lib/ask-answer";
import type { AssistantEntityRef } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { MentionKindTab, type MentionKind } from "@trenova/shared/types/generated/record-kinds";
import { useId, useMemo, useRef, useState, type FormEvent, type KeyboardEvent } from "react";
import {
  DeskMentionLoading,
  DeskMentionOption,
  MentionGlyph,
  useMentionSearch,
} from "../composer/desk-mentions";
import { deskSmallIconClass } from "../desk-button-styles";
import { DeskCardButton } from "../desk-card";
import { DeskIcon } from "../desk-icons";

/** The browser control each kind of typed value is entered with. */
const INPUT_TYPES = {
  text: "text",
  number: "number",
  date: "date",
  datetime: "datetime-local",
} as const;

/**
 * The values a question asks for, as one small form sent as one answer.
 *
 * A form of several values is sent as one line per value, `Label: value`, so
 * the agent reads which is which; a question that takes a single value of the
 * person's own (a number, a day, a record) is the same form with one field,
 * and sends the value alone. Either way it leaves as an ordinary message, and
 * Send waits until every value the form needs is there.
 */
export function DeskAskForm({
  fields,
  single = false,
  disabled,
  onAnswer,
}: {
  fields: readonly AskField[];
  /** The form is one value of the person's own: it sends the value without its label. */
  single?: boolean;
  disabled: boolean;
  onAnswer: (text: string, mentions: readonly AssistantEntityRef[]) => void;
}) {
  const t = useT();
  const id = useId();
  const [values, setValues] = useState<Record<string, AskValue>>({});
  const answer: AskAnswer | null =
    single && fields.length === 1
      ? askInputAnswer(fields[0], values[fields[0].key] ?? null)
      : askFieldsAnswer(fields, values);

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (answer === null || disabled) {
      return;
    }
    onAnswer(answer.text, answer.mentions);
  };

  return (
    <form className="dk-ask-form" onSubmit={submit} noValidate>
      {fields.map((field, index) => (
        <DeskAskFieldRow
          key={field.key}
          id={`${id}-${index}`}
          field={field}
          value={values[field.key] ?? null}
          disabled={disabled}
          onChange={(value) => setValues((current) => ({ ...current, [field.key]: value }))}
        />
      ))}
      <div className="dk-ask-send">
        <DeskCardButton ink type="submit" disabled={disabled || answer === null}>
          <DeskIcon name="enter" size={12} stroke={2} />
          {t("Send")}
        </DeskCardButton>
      </div>
    </form>
  );
}

/** One value of the form: its label, the control its kind takes, and what it should look like. */
function DeskAskFieldRow({
  id,
  field,
  value,
  disabled,
  onChange,
}: {
  id: string;
  field: AskField;
  value: AskValue;
  disabled: boolean;
  onChange: (value: AskValue) => void;
}) {
  const t = useT();
  const hintId = field.hint !== "" ? `${id}-hint` : undefined;

  return (
    <div className="dk-ask-f">
      <label className="dk-ask-fl" htmlFor={id}>
        {field.label}
        {field.optional && <em>{t("Optional")}</em>}
      </label>
      <div className="dk-ask-fc">
        {field.type === "record" ? (
          <DeskAskRecordPicker
            id={id}
            label={field.label}
            kind={field.recordKind}
            value={typeof value === "string" ? null : value}
            disabled={disabled}
            required={!field.optional}
            describedBy={hintId}
            onChange={onChange}
          />
        ) : (
          <input
            id={id}
            className="ui-field ui-focus-ring dk-ask-in"
            type={INPUT_TYPES[field.type]}
            inputMode={field.type === "number" ? "decimal" : undefined}
            step={field.type === "number" ? "any" : undefined}
            autoComplete="off"
            value={typeof value === "string" ? value : ""}
            disabled={disabled}
            required={!field.optional}
            aria-describedby={hintId}
            onChange={(event) => onChange(event.target.value)}
          />
        )}
        {hintId && (
          <span id={hintId} className="dk-ask-fh">
            {field.hint}
          </span>
        )}
      </div>
    </div>
  );
}

/**
 * Picking a record of one kind, from the same search the composer's @ runs.
 * Typing, a click or the down arrow opens it; the arrows walk what it found
 * and Enter takes one. A record taken shows as the chip a mention draws, with
 * a way to clear it.
 */
function DeskAskRecordPicker({
  id,
  label,
  kind,
  value,
  disabled,
  required,
  describedBy,
  onChange,
}: {
  id: string;
  label: string;
  kind: MentionKind;
  value: AssistantEntityRef | null;
  disabled: boolean;
  required: boolean;
  describedBy?: string;
  onChange: (value: AssistantEntityRef | null) => void;
}) {
  const t = useT();
  const listId = useId();
  const inputRef = useRef<HTMLInputElement>(null);
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const { results, loading, setLoading, highlighted, setHighlighted } = useMentionSearch(
    open ? query : null,
    MentionKindTab[kind],
  );
  // A tab can hold more than one kind (an invoice and one waiting to be
  // billed); only the kind asked for answers the question.
  const found = useMemo(() => results.filter((record) => record.type === kind), [kind, results]);
  const active = Math.min(highlighted, found.length - 1);

  const search = (next: string) => {
    setQuery(next);
    setLoading(true);
    setOpen(true);
  };

  const pick = (index: number) => {
    const record = found[index];
    if (!record) {
      return;
    }
    onChange({ type: record.type, id: record.id, label: record.label });
    setQuery("");
    setOpen(false);
  };

  const clear = () => {
    onChange(null);
    requestAnimationFrame(() => inputRef.current?.focus());
  };

  const onKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      if (!open) {
        search(query);
        return;
      }
      const step = event.key === "ArrowDown" ? 1 : -1;
      setHighlighted(Math.max(0, Math.min(found.length - 1, active + step)));
      return;
    }
    // Enter takes the record the keyboard is on; it never sends the form
    // from a search that has not found anything yet.
    if (event.key === "Enter") {
      event.preventDefault();
      if (open && !loading) {
        pick(active);
      }
      return;
    }
    if (event.key === "Escape" && open) {
      event.preventDefault();
      setOpen(false);
    }
  };

  if (value !== null) {
    return (
      <div className="ui-field dk-ask-in dk-ask-pick" aria-describedby={describedBy}>
        <span className="dk-mn-chip">
          <MentionGlyph type={value.type} size={12} />
          {value.label}
        </span>
        <Button
          id={id}
          variant="bare"
          size="bare"
          className={deskSmallIconClass}
          aria-label={t("Clear {0}", label)}
          disabled={disabled}
          onClick={clear}
        >
          <DeskIcon name="x" size={12} stroke={2} />
        </Button>
      </div>
    );
  }

  const activeId = open && !loading && found[active] ? `${listId}-${active}` : undefined;

  return (
    <>
      <input
        ref={inputRef}
        id={id}
        className="ui-field ui-focus-ring dk-ask-in"
        type="text"
        role="combobox"
        aria-expanded={open}
        aria-controls={listId}
        aria-autocomplete="list"
        aria-activedescendant={activeId}
        aria-describedby={describedBy}
        autoComplete="off"
        placeholder={t("Type to search")}
        value={query}
        disabled={disabled}
        required={required}
        onClick={() => {
          if (!open) {
            search(query);
          }
        }}
        onBlur={() => setOpen(false)}
        onChange={(event) => search(event.target.value)}
        onKeyDown={onKeyDown}
      />
      {open && (
        <div className="dk-mn dk-ask-mn" onMouseDown={(event) => event.preventDefault()}>
          <div className="dk-mn-l" role="listbox" id={listId} aria-label={label}>
            {loading && <DeskMentionLoading />}
            {!loading &&
              found.map((record, index) => (
                <DeskMentionOption
                  key={record.id}
                  id={`${listId}-${index}`}
                  record={record}
                  highlighted={index === active}
                  onHighlight={() => setHighlighted(index)}
                  onPick={() => pick(index)}
                />
              ))}
            {!loading && found.length === 0 && (
              <div className="dk-mn-empty">
                {query !== "" ? t("No records match “{0}”", query) : t("Start typing to find one")}
              </div>
            )}
          </div>
        </div>
      )}
    </>
  );
}
