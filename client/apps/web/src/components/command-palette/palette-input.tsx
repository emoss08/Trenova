import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { dropdownMenuSurfaceClassName } from "@trenova/shared/components/ui/dropdown-menu";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Command as CommandPrimitive } from "cmdk";
import { SearchLgIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { paletteEntityText } from "./palette-entities";
import type { PaletteScope } from "./palette-model";
import { isRecordScope } from "./palette-model";
import { PaletteTile } from "./palette-parts";
import { scopePresentation } from "./palette-scope-tabs";
import type { SearchEntityOption } from "./search-entity-filter";

function Chip({
  children,
  onRemove,
  removeLabel,
}: {
  children: React.ReactNode;
  onRemove?: () => void;
  removeLabel?: string;
}) {
  return (
    <span className="animate-pop-in bg-sunken text-foreground ring-border-subtle inline-flex h-6 max-w-56 shrink-0 items-center gap-1.5 rounded-full pr-1 pl-2 text-sm font-medium ring-1 ring-inset">
      {children}
      {onRemove && (
        <button
          type="button"
          tabIndex={-1}
          aria-label={removeLabel}
          onMouseDown={(event) => event.preventDefault()}
          onClick={onRemove}
          className="text-foreground-subtle hover:text-foreground flex size-4 items-center justify-center rounded-full transition-colors"
        >
          <XCloseIcon className="size-3" />
        </button>
      )}
    </span>
  );
}

export function scopePlaceholder(scope: PaletteScope, t: TranslateFn): string {
  if (isRecordScope(scope)) {
    return paletteEntityText(scope, t).placeholder;
  }
  switch (scope) {
    case "pages":
      return t("Jump to a page…");
    case "commands":
      return t("Run a command…");
    default:
      return t("Search records and pages, run a command, or ask Nova…");
  }
}

export interface PaletteInputProps {
  ref?: React.Ref<HTMLInputElement>;
  value: string;
  onValueChange: (value: string) => void;
  onKeyDown: (event: React.KeyboardEvent<HTMLInputElement>) => void;
  onClear: () => void;
  scope: PaletteScope;
  onClearScope: () => void;
  /** The query starts with ">", so only commands are searched. */
  commandMode: boolean;
  askSubject: string | null;
  onClearAskSubject: () => void;
  mention: {
    open: boolean;
    options: readonly SearchEntityOption[];
    index: number;
    onPick: (option: SearchEntityOption) => void;
  };
}

/**
 * The search box and what currently narrows it: a scope, the record a
 * question is about, or command mode. Each shows as a chip inside the box,
 * so what the palette is doing is always written where the person is looking.
 */
export function PaletteInput({
  ref,
  value,
  onValueChange,
  onKeyDown,
  onClear,
  scope,
  onClearScope,
  commandMode,
  askSubject,
  onClearAskSubject,
  mention,
}: PaletteInputProps) {
  const t = useT();
  const scoped = askSubject === null && scope !== "all" ? scopePresentation(scope, t) : null;

  const placeholder = askSubject
    ? t("Ask Nova about {0}…", askSubject)
    : scopePlaceholder(commandMode ? "commands" : scope, t);

  return (
    <div className="relative">
      <div className="flex h-13.5 items-center gap-2.5 px-4">
        <SearchLgIcon className="text-foreground-faint size-4.25 shrink-0" strokeWidth={1.75} />
        {askSubject && (
          <Chip onRemove={onClearAskSubject} removeLabel={t("Stop asking about this")}>
            <AssistMark className="size-3 shrink-0" />
            <span className="truncate">{askSubject}</span>
          </Chip>
        )}
        {scoped && (
          <Chip onRemove={onClearScope} removeLabel={t("Search everything")}>
            <scoped.icon className="text-foreground-subtle size-3 shrink-0" />
            {scoped.label}
          </Chip>
        )}
        {!askSubject && !scoped && commandMode && <Chip>{t("Commands")}</Chip>}
        <CommandPrimitive.Input
          ref={ref}
          value={value}
          onValueChange={onValueChange}
          onKeyDown={onKeyDown}
          placeholder={placeholder}
          autoFocus
          spellCheck={false}
          autoComplete="off"
          className="text-foreground placeholder:text-foreground-subtle h-full min-w-0 flex-1 bg-transparent text-lg outline-hidden"
        />
        {value !== "" && (
          <button
            type="button"
            tabIndex={-1}
            aria-label={t("Clear")}
            onMouseDown={(event) => event.preventDefault()}
            onClick={onClear}
            className="text-foreground-subtle hover:bg-surface-hover hover:text-foreground rounded-control flex size-6 shrink-0 items-center justify-center transition-colors"
          >
            <XCloseIcon className="size-3.5" />
          </button>
        )}
        <Kbd className="shrink-0">Esc</Kbd>
      </div>
      {mention.open && mention.options.length > 0 && (
        <div
          role="listbox"
          aria-label={t("Search one kind of thing")}
          className={cn(
            dropdownMenuSurfaceClassName,
            "animate-rise ui-lift-float absolute top-11 left-9 z-20 w-60 p-1",
          )}
        >
          {mention.options.map((option, index) => {
            const presentation = scopePresentation(option.key, t);
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
                  "rounded-control text-foreground flex h-8 w-full items-center gap-2 px-2 text-left text-base transition-colors",
                  index === mention.index ? "bg-surface-hover" : "hover:bg-surface-hover",
                )}
              >
                <PaletteTile icon={presentation.icon} size="chip" />
                <span className="flex-1 truncate">{presentation.label}</span>
                <span className="text-foreground-faint font-mono text-xs">@{option.key}</span>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
