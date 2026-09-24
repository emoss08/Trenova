import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Command as CommandPrimitive } from "cmdk";
import { ArrowLeftIcon, SearchIcon, XIcon } from "lucide-react";
import { PALETTE_ENTITIES } from "./palette-entities";
import type { PaletteScope } from "./palette-model";
import { isRecordScope } from "./palette-model";
import type { SearchEntityOption } from "./search-entity-filter";

function Chip({
  children,
  onRemove,
  removeLabel,
  className,
}: {
  children: React.ReactNode;
  onRemove: () => void;
  removeLabel: string;
  className?: string;
}) {
  return (
    <span
      className={cn(
        "inline-flex h-6 shrink-0 items-center gap-1 rounded-full pr-1 pl-2 text-xs font-medium ring-1 ring-inset",
        className,
      )}
    >
      {children}
      <button
        type="button"
        tabIndex={-1}
        aria-label={removeLabel}
        onMouseDown={(event) => event.preventDefault()}
        onClick={onRemove}
        className="flex size-4 items-center justify-center rounded-full opacity-70 transition-opacity hover:opacity-100"
      >
        <XIcon className="size-3" />
      </button>
    </span>
  );
}

function scopePlaceholder(scope: PaletteScope, t: TranslateFn): string {
  switch (scope) {
    case "shipment":
      return t("Search shipments by PRO or BOL…");
    case "customer":
      return t("Search customers by name or code…");
    case "worker":
      return t("Search workers by name…");
    case "document":
      return t("Search documents by file name…");
    case "pages":
      return t("Jump to a page…");
    case "commands":
      return t("Run a command…");
    case "all":
      return t("Search records, pages and commands, or ask a question…");
  }
}

export interface PaletteInputProps {
  ref?: React.Ref<HTMLInputElement>;
  value: string;
  onValueChange: (value: string) => void;
  onKeyDown: (event: React.KeyboardEvent<HTMLInputElement>) => void;
  scope: PaletteScope;
  onClearScope: () => void;
  /** The row whose actions are listed, when the action view is open. */
  actionsFor: string | null;
  onBack: () => void;
  askSubject: string | null;
  onClearAskSubject: () => void;
  busy: boolean;
  mention: {
    open: boolean;
    options: readonly SearchEntityOption[];
    index: number;
    onPick: (option: SearchEntityOption) => void;
  };
}

/**
 * The search box and what currently narrows it: a record scope, the record
 * a question is about, or the row whose actions are listed. Each shows as a
 * removable chip inside the box, so what the palette is doing is always
 * written where the person is looking.
 */
export function PaletteInput({
  ref,
  value,
  onValueChange,
  onKeyDown,
  scope,
  onClearScope,
  actionsFor,
  onBack,
  askSubject,
  onClearAskSubject,
  busy,
  mention,
}: PaletteInputProps) {
  const t = useT();
  const recordScope = isRecordScope(scope) ? PALETTE_ENTITIES[scope] : null;

  const placeholder = actionsFor
    ? t("Search actions…")
    : askSubject
      ? t("Ask anything about it…")
      : scopePlaceholder(scope, t);

  return (
    <div className="relative">
      <div className="flex h-14 items-center gap-2.5 px-4">
        {actionsFor ? (
          <button
            type="button"
            tabIndex={-1}
            aria-label={t("Back to results")}
            onMouseDown={(event) => event.preventDefault()}
            onClick={onBack}
            className="ui-press rounded-control text-foreground-muted hover:bg-surface-hover hover:text-foreground flex size-6 shrink-0 items-center justify-center transition-colors"
          >
            <ArrowLeftIcon className="size-4" />
          </button>
        ) : busy ? (
          <Spinner className="text-foreground-subtle size-4 shrink-0" />
        ) : (
          <SearchIcon className="text-foreground-subtle size-4 shrink-0" strokeWidth={1.75} />
        )}
        {actionsFor && (
          <span className="bg-sunken text-foreground-muted ring-border-subtle max-w-48 shrink-0 truncate rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset">
            {actionsFor}
          </span>
        )}
        {!actionsFor && recordScope && (
          <Chip
            onRemove={onClearScope}
            removeLabel={t("Search everything")}
            className={recordScope.tileClass}
          >
            <recordScope.icon className="size-3" />
            {t(recordScope.pluralLabel)}
          </Chip>
        )}
        {!actionsFor && askSubject && (
          <Chip
            onRemove={onClearAskSubject}
            removeLabel={t("Stop asking about this")}
            className="bg-brand-subtle text-brand-subtle-foreground ring-brand-border"
          >
            <AssistMark className="size-3" />
            <span className="max-w-40 truncate">{askSubject}</span>
          </Chip>
        )}
        <CommandPrimitive.Input
          ref={ref}
          value={value}
          onValueChange={onValueChange}
          onKeyDown={onKeyDown}
          placeholder={placeholder}
          autoFocus
          className="text-foreground placeholder:text-foreground-subtle h-full min-w-0 flex-1 bg-transparent text-base outline-hidden"
        />
      </div>
      {mention.open && mention.options.length > 0 && (
        <div
          role="listbox"
          aria-label={t("Search one kind of record")}
          className="animate-rise rounded-surface bg-raised ring-foreground/10 ui-lift-float absolute top-12 left-10 z-10 w-60 p-1 ring-1"
        >
          <p className="text-foreground-subtle px-2 pt-1 pb-1.5 text-xs">{t("Search only…")}</p>
          {mention.options.map((option, index) => {
            const entity = PALETTE_ENTITIES[option.key];
            return (
              <button
                key={option.key}
                type="button"
                role="option"
                aria-selected={index === mention.index}
                tabIndex={-1}
                onMouseDown={(event) => event.preventDefault()}
                onClick={() => mention.onPick(option)}
                className={cn(
                  "rounded-control flex w-full items-center gap-2 px-2 py-1.5 text-left text-sm transition-colors",
                  index === mention.index ? "bg-surface-selected" : "hover:bg-surface-hover",
                )}
              >
                <span
                  className={cn(
                    "rounded-control flex size-5 items-center justify-center ring-1 ring-inset",
                    entity.tileClass,
                  )}
                >
                  <entity.icon className="size-3" />
                </span>
                <span className="flex-1">{t(option.label)}</span>
                <span className="text-2xs text-foreground-subtle font-mono">
                  @{option.aliases[0]}
                </span>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
