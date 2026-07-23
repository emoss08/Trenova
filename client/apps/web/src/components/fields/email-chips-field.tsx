import { FieldWrapper } from "@/components/fields/field-components";
import { cn } from "@trenova/shared/lib/utils";
import { XIcon } from "lucide-react";
import { useState } from "react";
import {
  Controller,
  type Control,
  type FieldPath,
  type FieldValues,
  type RegisterOptions,
} from "react-hook-form";

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

function splitCandidates(raw: string): string[] {
  return raw
    .split(/[,;\s]+/)
    .map((value) => value.trim())
    .filter(Boolean);
}

export function EmailChipsField<T extends FieldValues>({
  control,
  name,
  label,
  description,
  placeholder,
  rules,
  className,
}: {
  control: Control<T>;
  name: FieldPath<T>;
  label?: string;
  description?: string;
  placeholder?: string;
  rules?: RegisterOptions<T, FieldPath<T>>;
  className?: string;
}) {
  const [draft, setDraft] = useState("");
  const [invalidDraft, setInvalidDraft] = useState<string | null>(null);
  const inputId = `email-chips-${name}`;

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => {
        const emails: string[] = field.value ?? [];

        const commit = (raw: string): boolean => {
          const candidates = splitCandidates(raw);
          if (candidates.length === 0) return true;

          const invalid = candidates.find((value) => !EMAIL_PATTERN.test(value));
          if (invalid) {
            setInvalidDraft(invalid);
            return false;
          }

          const merged = [...emails];
          for (const candidate of candidates) {
            const normalized = candidate.toLowerCase();
            if (!merged.some((existing) => existing.toLowerCase() === normalized)) {
              merged.push(candidate);
            }
          }
          field.onChange(merged);
          setDraft("");
          setInvalidDraft(null);
          return true;
        };

        const remove = (email: string) => {
          field.onChange(emails.filter((value) => value !== email));
        };

        return (
          <FieldWrapper
            label={label}
            required={!!rules?.required}
            description={description}
            error={invalidDraft ? `"${invalidDraft}" is not a valid email address` : fieldState.error?.message}
            className={className}
          >
            <label
              htmlFor={inputId}
              className={cn(
                "flex min-h-7 flex-wrap items-center gap-1 rounded-md border border-input bg-muted px-1.5 py-1",
                "cursor-text transition-[border-color,box-shadow] duration-200 ease-in-out",
                "focus-within:border-brand focus-within:ring-4 focus-within:ring-brand/30",
                (invalidDraft || fieldState.invalid) &&
                  "border-destructive bg-destructive/20 focus-within:border-destructive focus-within:ring-destructive/20",
              )}
            >
              {emails.map((email) => (
                <span
                  key={email}
                  className="inline-flex max-w-full items-center gap-1 rounded-sm border border-border bg-background py-0.5 pr-1 pl-1.5 text-xs"
                >
                  <span className="truncate">{email}</span>
                  <button
                    type="button"
                    aria-label={`Remove ${email}`}
                    className="rounded-xs text-muted-foreground transition-colors hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
                    onClick={(event) => {
                      event.stopPropagation();
                      remove(email);
                    }}
                  >
                    <XIcon className="size-3" />
                  </button>
                </span>
              ))}
              <input
                id={inputId}
                type="text"
                inputMode="email"
                autoComplete="off"
                spellCheck={false}
                value={draft}
                placeholder={emails.length === 0 ? placeholder : undefined}
                className="min-w-24 flex-1 bg-transparent py-0.5 text-xs outline-none placeholder:text-muted-foreground"
                onChange={(event) => {
                  setDraft(event.target.value);
                  if (invalidDraft) setInvalidDraft(null);
                }}
                onKeyDown={(event) => {
                  if (event.key === "Enter" || event.key === ",") {
                    event.preventDefault();
                    commit(draft);
                  } else if (event.key === "Backspace" && draft === "" && emails.length > 0) {
                    remove(emails[emails.length - 1]);
                  }
                }}
                onBlur={() => {
                  commit(draft);
                  field.onBlur();
                }}
                onPaste={(event) => {
                  event.preventDefault();
                  const pasted = event.clipboardData.getData("text");
                  commit(draft ? `${draft} ${pasted}` : pasted);
                }}
              />
            </label>
          </FieldWrapper>
        );
      }}
    />
  );
}
