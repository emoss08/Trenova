"use no memo";
import { Input } from "@trenova/shared/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { Switch } from "@trenova/shared/components/ui/switch";
import { fromUserWallClock, toUserWallClock } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { Cell, FilterVariant } from "@trenova/shared/types/data-table";
import type { SelectOption } from "@trenova/shared/types/fields";
import { useCallback, useRef, useState } from "react";
import { toast } from "sonner";

function pad(value: number): string {
  return String(value).padStart(2, "0");
}

function unixToDateTimeLocal(value: unknown): string {
  if (typeof value !== "number" || !Number.isFinite(value)) return "";
  const wallClock = toUserWallClock(value);
  if (!wallClock) return "";
  return `${wallClock.getFullYear()}-${pad(wallClock.getMonth() + 1)}-${pad(wallClock.getDate())}T${pad(wallClock.getHours())}:${pad(wallClock.getMinutes())}`;
}

function dateTimeLocalToUnix(value: string): number | null {
  if (!value) return null;
  const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})/.exec(value);
  if (!match) return null;
  const wallClock = new Date(
    Number(match[1]),
    Number(match[2]) - 1,
    Number(match[3]),
    Number(match[4]),
    Number(match[5]),
  );
  return fromUserWallClock(wallClock) ?? null;
}

function initialDraftFor(variant: FilterVariant, value: unknown): string {
  switch (variant) {
    case "number":
      return typeof value === "number" && Number.isFinite(value) ? String(value) : "";
    case "date":
      return unixToDateTimeLocal(value);
    default:
      return value === null || value === undefined ? "" : String(value);
  }
}

function draftToValue(variant: FilterVariant, draft: string): unknown {
  switch (variant) {
    case "number": {
      if (draft.trim() === "") return null;
      const parsed = Number(draft);
      return Number.isFinite(parsed) ? parsed : null;
    }
    case "date":
      return dateTimeLocalToUnix(draft);
    default: {
      const trimmed = draft.trim();
      return trimmed === "" ? null : trimmed;
    }
  }
}

type DataTableCellEditorProps<TData extends Record<string, any>> = {
  cell: Cell<TData, unknown>;
};

export function DataTableCellEditor<TData extends Record<string, any>>({
  cell,
}: DataTableCellEditorProps<TData>) {
  "use no memo";
  const variant = (cell.column.columnDef.meta?.filterType ?? "text") as FilterVariant;
  const filterOptions = cell.column.columnDef.meta?.filterOptions as SelectOption[] | undefined;
  const initialValue = cell.getValue();
  const [draft, setDraft] = useState(() => initialDraftFor(variant, initialValue));
  const [isPending, setIsPending] = useState(false);
  const cancelledRef = useRef(false);

  const commitValue = useCallback(
    async (value: unknown) => {
      if (Object.is(value, initialValue) || (value === null && initialValue === undefined)) {
        cell.stopEditing();
        return;
      }
      setIsPending(true);
      try {
        await cell.commitEdit(value);
      } catch (error) {
        toast.error("Update failed", {
          description: error instanceof Error ? error.message : "The change could not be saved.",
        });
        setIsPending(false);
      }
    },
    [cell, initialValue],
  );

  const handleInputKeyDown = useCallback(
    (event: React.KeyboardEvent<HTMLInputElement>) => {
      if (event.key === "Enter") {
        event.preventDefault();
        void commitValue(draftToValue(variant, event.currentTarget.value));
      } else if (event.key === "Escape") {
        event.preventDefault();
        event.stopPropagation();
        cancelledRef.current = true;
        cell.stopEditing();
      }
    },
    [cell, commitValue, variant],
  );

  const handleInputBlur = useCallback(
    (event: React.FocusEvent<HTMLInputElement>) => {
      if (cancelledRef.current || isPending) return;
      void commitValue(draftToValue(variant, event.currentTarget.value));
    },
    [commitValue, isPending, variant],
  );

  const stopClickThrough = useCallback((event: React.MouseEvent) => {
    event.stopPropagation();
  }, []);

  if (variant === "boolean") {
    return (
      <span className="flex items-center gap-2" onClick={stopClickThrough}>
        <Switch
          autoFocus
          disabled={isPending}
          checked={initialValue === true}
          onCheckedChange={(checked) => void commitValue(checked)}
          onKeyDown={(event) => {
            if (event.key === "Escape") {
              event.stopPropagation();
              cell.stopEditing();
            }
          }}
          onBlur={() => {
            if (!isPending) cell.stopEditing();
          }}
          aria-label={`Edit ${cell.column.id}`}
        />
        {isPending && <Spinner className="size-3.5" />}
      </span>
    );
  }

  if (variant === "select") {
    return (
      <span className="flex w-full items-center gap-2" onClick={stopClickThrough}>
        <Select
          defaultOpen
          disabled={isPending}
          value={initialValue === null || initialValue === undefined ? "" : String(initialValue)}
          onValueChange={(value) => void commitValue(value)}
          onOpenChange={(open) => {
            if (!open && !isPending) cell.stopEditing();
          }}
        >
          <SelectTrigger
            size="sm"
            className="h-7 w-full min-w-0 text-xs"
            aria-label={`Edit ${cell.column.id}`}
          >
            <SelectValue placeholder="Select a value" />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              {(filterOptions ?? []).map((option) => (
                <SelectItem key={String(option.value)} value={String(option.value)}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
        {isPending && <Spinner className="size-3.5 shrink-0" />}
      </span>
    );
  }

  return (
    <span className="flex w-full items-center gap-2" onClick={stopClickThrough}>
      <Input
        autoFocus
        disabled={isPending}
        type={variant === "number" ? "number" : variant === "date" ? "datetime-local" : "text"}
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
        onKeyDown={handleInputKeyDown}
        onBlur={handleInputBlur}
        className={cn("h-7 w-full min-w-0 px-2 text-xs")}
        aria-label={`Edit ${cell.column.id}`}
      />
      {isPending && <Spinner className="size-3.5 shrink-0" />}
    </span>
  );
}
