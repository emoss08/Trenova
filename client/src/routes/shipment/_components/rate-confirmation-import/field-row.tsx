import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { CheckIcon, PencilIcon, UndoIcon, ChevronDownIcon } from "lucide-react";
import { useCallback, useRef, useState } from "react";
import { type ReconciliationField, type FieldStatus, getEffectiveValue } from "./types";

type FieldRowProps = {
  field: ReconciliationField;
  onAccept: (key: string) => void;
  onEdit: (key: string, value: unknown) => void;
  onReset: (key: string) => void;
  onSelectAlternative?: (key: string, value: string) => void;
};

const STATUS_STYLES: Record<FieldStatus, { dot: string; bg: string }> = {
  accepted: { dot: "bg-emerald-500", bg: "" },
  "needs-review": { dot: "bg-amber-500", bg: "bg-amber-500/[0.04]" },
  missing: { dot: "bg-muted-foreground/20", bg: "" },
  conflicting: { dot: "bg-amber-500", bg: "bg-amber-500/[0.04]" },
  edited: { dot: "bg-blue-500", bg: "bg-blue-500/[0.04]" },
};

function displayValue(value: unknown): string {
  if (value == null) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number") return String(value);
  return JSON.stringify(value);
}

export function FieldRow({ field, onAccept, onEdit, onReset, onSelectAlternative }: FieldRowProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [editValue, setEditValue] = useState("");
  const [showAlts, setShowAlts] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const style = STATUS_STYLES[field.status];
  const effectiveValue = getEffectiveValue(field);
  const displayVal = displayValue(effectiveValue);

  const handleStartEdit = useCallback(() => {
    setEditValue(displayVal);
    setIsEditing(true);
    requestAnimationFrame(() => inputRef.current?.focus());
  }, [displayVal]);

  const handleSaveEdit = useCallback(() => {
    setIsEditing(false);
    if (editValue !== displayVal) {
      onEdit(field.key, editValue);
    }
  }, [editValue, displayVal, field.key, onEdit]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter") {
        if (isEditing) handleSaveEdit();
        else if (field.status === "needs-review" || field.status === "conflicting") onAccept(field.key);
      }
      if (e.key === "Escape") {
        if (isEditing) setIsEditing(false);
        else if (field.status === "edited") onReset(field.key);
      }
    },
    [isEditing, handleSaveEdit, field.status, field.key, onAccept, onReset],
  );

  // Compact single-line for missing fields
  if (field.status === "missing" && !isEditing) {
    return (
      <div
        className="group flex items-center gap-2 rounded px-2 py-1 transition-colors hover:bg-muted/50"
        onKeyDown={handleKeyDown}
        tabIndex={0}
        role="row"
      >
        <div className={cn("size-1.5 shrink-0 rounded-full", style.dot)} />
        <span className="text-xs text-muted-foreground/50">{field.label}</span>
        <span className="text-xs text-muted-foreground/30 italic">Not extracted</span>
        <Button
          variant="ghost"
          size="icon-xs"
          className="ml-auto opacity-0 transition-opacity group-hover:opacity-100"
          onClick={handleStartEdit}
        >
          <PencilIcon className="size-2.5" />
        </Button>
      </div>
    );
  }

  return (
    <div
      className={cn(
        "group rounded px-2 py-1.5 transition-colors",
        style.bg,
        !style.bg && "hover:bg-muted/30",
      )}
      onKeyDown={handleKeyDown}
      tabIndex={0}
      role="row"
    >
      <div className="flex items-start gap-2">
        <div className={cn("mt-1.5 size-1.5 shrink-0 rounded-full", style.dot)} />
        <div className="min-w-0 flex-1">
          <div className="flex items-center justify-between gap-2">
            <div className="flex min-w-0 items-baseline gap-2">
              <span className="shrink-0 text-2xs text-muted-foreground">{field.label}</span>
              {isEditing ? (
                <Input
                  ref={inputRef}
                  value={editValue}
                  onChange={(e) => setEditValue(e.target.value)}
                  onBlur={handleSaveEdit}
                  className="h-6 flex-1 text-xs"
                />
              ) : (
                <span className="truncate text-xs text-foreground">{displayVal}</span>
              )}
            </div>
            <div className="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-focus-within:opacity-100 group-hover:opacity-100">
              {(field.status === "needs-review" || field.status === "conflicting") && (
                <Button variant="ghost" size="icon-xs" onClick={() => onAccept(field.key)}>
                  <CheckIcon className="size-2.5" />
                </Button>
              )}
              {!isEditing && (
                <Button variant="ghost" size="icon-xs" onClick={handleStartEdit}>
                  <PencilIcon className="size-2.5" />
                </Button>
              )}
              {field.status === "edited" && (
                <Button variant="ghost" size="icon-xs" onClick={() => onReset(field.key)}>
                  <UndoIcon className="size-2.5" />
                </Button>
              )}
            </div>
          </div>

          {/* Evidence — constrained width, truncated */}
          {field.evidenceExcerpt && !isEditing && (
            <p className="mt-0.5 max-w-[320px] truncate text-2xs text-muted-foreground/40">
              &ldquo;{field.evidenceExcerpt}&rdquo;
              {field.pageNumber != null && <span className="ml-1">p.{field.pageNumber}</span>}
            </p>
          )}

          {/* Alternatives */}
          {field.alternativeValues && field.alternativeValues.length > 1 && !isEditing && (
            <div className="mt-0.5">
              <button
                type="button"
                className="flex items-center gap-1 text-2xs text-muted-foreground/50 transition-colors hover:text-muted-foreground"
                onClick={() => setShowAlts(!showAlts)}
              >
                <ChevronDownIcon className={cn("size-2.5 transition-transform", showAlts && "rotate-180")} />
                {field.alternativeValues.length} alternatives
              </button>
              {showAlts && (
                <div className="mt-1 flex flex-wrap gap-1">
                  {field.alternativeValues.map((alt) => (
                    <button
                      key={alt}
                      type="button"
                      className="rounded border px-1.5 py-0.5 text-2xs transition-colors hover:bg-muted"
                      onClick={() => {
                        onSelectAlternative?.(field.key, alt);
                        setShowAlts(false);
                      }}
                    >
                      {alt}
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
