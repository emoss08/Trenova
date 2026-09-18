import { FieldWrapper } from "@/components/fields/field-components";
import { cn } from "@trenova/shared/lib/utils";
import { XIcon } from "lucide-react";
import { useId, useState } from "react";
import {
  Controller,
  type Control,
  type FieldPath,
  type FieldValues,
  type RegisterOptions,
} from "react-hook-form";

type TextChipsFieldProps<T extends FieldValues> = {
  control: Control<T>;
  name: FieldPath<T>;
  label?: string;
  description?: string;
  placeholder?: string;
  rules?: RegisterOptions<T, FieldPath<T>>;
  className?: string;
  /** Longest entry accepted; longer ones are refused with a message. */
  maxLength?: number;
  /** Most entries accepted; the input closes once reached. */
  maxItems?: number;
};

/**
 * Free-text chips: Enter or a comma commits the draft, Backspace on an empty
 * box removes the last chip. Duplicates are ignored case-insensitively.
 */
export function TextChipsField<T extends FieldValues>({
  control,
  name,
  label,
  description,
  placeholder,
  rules,
  className,
  maxLength = 300,
  maxItems,
}: TextChipsFieldProps<T>) {
  const [draft, setDraft] = useState("");
  const [draftError, setDraftError] = useState<string | null>(null);
  const inputId = useId();

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => {
        const items: string[] = field.value ?? [];
        const full = maxItems !== undefined && items.length >= maxItems;

        const commit = (): boolean => {
          const value = draft.trim();
          if (value === "") return true;
          if (value.length > maxLength) {
            setDraftError(`Keep each entry under ${maxLength} characters`);
            return false;
          }
          if (full) {
            setDraftError(`At most ${maxItems} entries`);
            return false;
          }
          if (!items.some((existing) => existing.toLowerCase() === value.toLowerCase())) {
            field.onChange([...items, value]);
          }
          setDraft("");
          setDraftError(null);
          return true;
        };

        const remove = (value: string) => field.onChange(items.filter((item) => item !== value));

        return (
          <FieldWrapper
            label={label}
            required={!!rules?.required}
            description={description}
            error={draftError ?? fieldState.error?.message}
            className={className}
          >
            <label
              htmlFor={inputId}
              className={cn(
                "border-input bg-muted flex min-h-7 flex-wrap items-center gap-1 rounded-md border px-1.5 py-1",
                "cursor-text transition-[border-color,box-shadow] duration-200 ease-in-out",
"ui-container-focus-ring",
                (draftError || fieldState.invalid) &&
"ui-container-focus-ring [--ring:var(--ring-danger)] border-destructive bg-destructive/20",
              )}
            >
              {items.map((item) => (
                <span
                  key={item}
                  className="border-border bg-background inline-flex max-w-full items-center gap-1 rounded-sm border py-0.5 pr-1 pl-1.5 text-xs"
                >
                  <span className="truncate">{item}</span>
                  <button
                    type="button"
                    aria-label={`Remove ${item}`}
 className="ui-focus-ring text-muted-foreground hover:text-foreground rounded-xs transition-colors"
                    onClick={(event) => {
                      event.preventDefault();
                      remove(item);
                    }}
                  >
                    <XIcon className="size-3" />
                  </button>
                </span>
              ))}
              {!full && (
                <input
                  id={inputId}
                  value={draft}
                  placeholder={items.length === 0 ? placeholder : undefined}
                  onChange={(event) => {
                    setDraft(event.target.value);
                    setDraftError(null);
                  }}
                  onKeyDown={(event) => {
                    if (event.key === "Enter" || event.key === ",") {
                      event.preventDefault();
                      commit();
                    } else if (event.key === "Backspace" && draft === "" && items.length > 0) {
                      remove(items[items.length - 1]);
                    }
                  }}
                  onBlur={() => {
                    commit();
                    field.onBlur();
                  }}
                  className="placeholder:text-muted-foreground min-w-32 flex-1 bg-transparent px-1 py-0.5 text-xs outline-none"
                />
              )}
            </label>
          </FieldWrapper>
        );
      }}
    />
  );
}
