import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { PlusIcon, XIcon } from "lucide-react";
import { useCallback } from "react";
import {
  Controller,
  type Control,
  type FieldValues,
  type Path,
} from "react-hook-form";

type KeyValueEditorProps<T extends FieldValues> = {
  control: Control<T>;
  name: Path<T>;
  label?: string;
  description?: string;
  keyPlaceholder?: string;
  valuePlaceholder?: string;
  className?: string;
  disabled?: boolean;
};

export function KeyValueEditor<T extends FieldValues>({
  control,
  name,
  label,
  description,
  keyPlaceholder = "Key",
  valuePlaceholder = "Value",
  className,
  disabled,
}: KeyValueEditorProps<T>) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => (
        <KeyValueEditorInner
          value={(field.value as Record<string, string>) ?? {}}
          onChange={field.onChange}
          label={label}
          description={description}
          keyPlaceholder={keyPlaceholder}
          valuePlaceholder={valuePlaceholder}
          className={className}
          disabled={disabled}
          error={fieldState.error?.message}
        />
      )}
    />
  );
}

function KeyValueEditorInner({
  value,
  onChange,
  label,
  description,
  keyPlaceholder,
  valuePlaceholder,
  className,
  disabled,
  error,
}: {
  value: Record<string, string>;
  onChange: (val: Record<string, string>) => void;
  label?: string;
  description?: string;
  keyPlaceholder?: string;
  valuePlaceholder?: string;
  className?: string;
  disabled?: boolean;
  error?: string;
}) {
  const entries = Object.entries(value);

  const addEntry = useCallback(() => {
    const newKey = `field_${entries.length + 1}`;
    onChange({ ...value, [newKey]: "" });
  }, [value, entries.length, onChange]);

  const removeEntry = useCallback(
    (key: string) => {
      const next = Object.fromEntries(
        Object.entries(value).filter(([entryKey]) => entryKey !== key),
      );
      onChange(next);
    },
    [value, onChange],
  );

  const updateKey = useCallback(
    (oldKey: string, newKey: string) => {
      if (newKey === oldKey) return;
      const newObj: Record<string, string> = {};
      for (const [k, v] of Object.entries(value)) {
        newObj[k === oldKey ? newKey : k] = v;
      }
      onChange(newObj);
    },
    [value, onChange],
  );

  const updateValue = useCallback(
    (key: string, newVal: string) => {
      onChange({ ...value, [key]: newVal });
    },
    [value, onChange],
  );

  return (
    <div className={cn("space-y-1.5", className)}>
      {label && (
        <label className="text-sm leading-none font-medium">{label}</label>
      )}
      <div className="space-y-2">
        {entries.map(([key, val]) => (
          <div key={key} className="flex items-center gap-2">
            <Input
              value={key}
              onChange={(e) => updateKey(key, e.target.value)}
              placeholder={keyPlaceholder}
              disabled={disabled}
              className="flex-1"
            />
            <Input
              value={val}
              onChange={(e) => updateValue(key, e.target.value)}
              placeholder={valuePlaceholder}
              disabled={disabled}
              className="flex-1"
            />
            {!disabled && (
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={() => removeEntry(key)}
              >
                <XIcon className="size-4" />
              </Button>
            )}
          </div>
        ))}
        {!disabled && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={addEntry}
            className="gap-1"
          >
            <PlusIcon className="size-3.5" />
            Add Entry
          </Button>
        )}
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
