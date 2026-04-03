import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { XIcon } from "lucide-react";
import { type KeyboardEvent, useCallback, useRef, useState } from "react";
import {
  Controller,
  type Control,
  type FieldValues,
  type Path,
} from "react-hook-form";

type TagInputProps<T extends FieldValues> = {
  control: Control<T>;
  name: Path<T>;
  label?: string;
  description?: string;
  placeholder?: string;
  className?: string;
  disabled?: boolean;
};

export function TagInput<T extends FieldValues>({
  control,
  name,
  label,
  description,
  placeholder = "Type and press Enter",
  className,
  disabled,
}: TagInputProps<T>) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => (
        <TagInputInner
          value={(field.value as string[]) ?? []}
          onChange={field.onChange}
          label={label}
          description={description}
          placeholder={placeholder}
          className={className}
          disabled={disabled}
          error={fieldState.error?.message}
        />
      )}
    />
  );
}

function TagInputInner({
  value,
  onChange,
  label,
  description,
  placeholder,
  className,
  disabled,
  error,
}: {
  value: string[];
  onChange: (val: string[]) => void;
  label?: string;
  description?: string;
  placeholder?: string;
  className?: string;
  disabled?: boolean;
  error?: string;
}) {
  const [input, setInput] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  const addTag = useCallback(
    (tag: string) => {
      const trimmed = tag.trim();
      if (trimmed && !value.includes(trimmed)) {
        onChange([...value, trimmed]);
      }
      setInput("");
    },
    [value, onChange],
  );

  const removeTag = useCallback(
    (index: number) => {
      onChange(value.filter((_, i) => i !== index));
    },
    [value, onChange],
  );

  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Enter") {
        e.preventDefault();
        addTag(input);
      } else if (e.key === "Backspace" && !input && value.length > 0) {
        removeTag(value.length - 1);
      }
    },
    [input, value, addTag, removeTag],
  );

  return (
    <div className={cn("space-y-1.5", className)}>
      {label && (
        <label className="text-sm leading-none font-medium">{label}</label>
      )}
      <div
        className={cn(
          "flex min-h-9 flex-wrap items-center gap-1.5 rounded-md border border-input bg-muted px-2.5 py-1.5 text-sm transition-[border-color,box-shadow] duration-200 ease-in-out",
          "focus-within:border-brand focus-within:ring-4 focus-within:ring-brand/30",
          error && "border-destructive",
          disabled && "pointer-events-none opacity-50",
        )}
        onClick={() => inputRef.current?.focus()}
      >
        {value.map((tag, i) => (
          <Badge key={`${tag}-${i}`} variant="secondary" className="gap-1 pr-1">
            {tag}
            {!disabled && (
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  removeTag(i);
                }}
                className="rounded-full hover:bg-muted-foreground/20"
              >
                <XIcon className="size-3" />
              </button>
            )}
          </Badge>
        ))}
        <input
          ref={inputRef}
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          onBlur={() => input && addTag(input)}
          placeholder={value.length === 0 ? placeholder : ""}
          disabled={disabled}
          className="min-w-[80px] flex-1 border-0 bg-transparent p-0 text-sm outline-none placeholder:text-muted-foreground"
        />
      </div>
      {(error || description) && (
        <p
          className={cn(
            "text-2xs",
            error ? "text-destructive" : "text-muted-foreground",
          )}
        >
          {error || description}
        </p>
      )}
    </div>
  );
}
