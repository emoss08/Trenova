import { useT } from "@trenova/shared/i18n/use-t";
import { FieldWrapper } from "@/components/fields/field-components";
import { cn } from "@trenova/shared/lib/utils";
import { XCloseIcon } from "@trenova/shared/components/icons";
import { useId, useState } from "react";
import {
  Controller,
  type Control,
  type FieldPath,
  type FieldValues,
  type RegisterOptions,
} from "react-hook-form";
import { fieldInvalidClass } from "@trenova/shared/lib/variants/field";

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
  const t = useT();
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
            setDraftError(t("Keep each entry under {0} characters", maxLength));
            return false;
          }
          if (full) {
            setDraftError(t("At most {0} entries", maxItems));
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
            name={name}
            label={label}
            required={!!rules?.required}
            description={description}
            error={draftError ?? fieldState.error?.message}
            className={className}
          >
            <label
              htmlFor={inputId}
              className={cn(
                "ui-field flex min-h-7 flex-wrap items-center gap-1 px-1.5 py-1",
                "cursor-text",
                "ui-container-focus-ring",
                (draftError || fieldState.invalid) && fieldInvalidClass,
              )}
            >
              {items.map((item) => (
                <span
                  key={item}
                  className="border-border bg-card inline-flex max-w-full items-center gap-1 rounded-sm border py-0.5 pr-1 pl-1.5 text-xs"
                >
                  <span className="truncate">{item}</span>
                  <button
                    type="button"
                    aria-label={t("Remove {0}", item)}
                    className="ui-focus-ring text-muted-foreground hover:text-foreground rounded-xs transition-colors"
                    onClick={(event) => {
                      event.preventDefault();
                      remove(item);
                    }}
                  >
                    <XCloseIcon className="size-3" />
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
