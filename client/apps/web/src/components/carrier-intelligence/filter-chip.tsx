import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@trenova/shared/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { cn } from "@trenova/shared/lib/utils";
import { CheckIcon, ChevronLeftIcon, ListFilterIcon, XIcon } from "lucide-react";
import { useState, type ReactNode } from "react";

export type FilterChipProps = {
  label: ReactNode;
  value: ReactNode;
  onClear: () => void;
  onEdit?: () => void;
  className?: string;
};

export function FilterChip({ label, value, onClear, onEdit, className }: FilterChipProps) {
  const t = useT();
  return (
    <span
      className={cn(
        "inline-flex h-8 items-center overflow-hidden rounded-md border bg-background text-xs",
        className,
      )}
    >
      <button
        type="button"
        onClick={onEdit}
        disabled={!onEdit}
        className="flex h-full items-center gap-1.5 px-2 enabled:hover:bg-muted"
      >
        <span className="text-muted-foreground">{label}</span>
        <span className="font-medium">{value}</span>
      </button>
      <button
        type="button"
        onClick={onClear}
        aria-label={t("Clear filter")}
        className="flex h-full items-center border-l px-1.5 text-muted-foreground hover:bg-muted hover:text-foreground"
      >
        <XIcon className="size-3" aria-hidden="true" />
      </button>
    </span>
  );
}

export type FilterOption = {
  value: string;
  label: ReactNode;
  keywords?: string;
  icon?: ReactNode;
};

export type FilterDefinition = {
  id: string;
  label: string;
  icon?: ReactNode;
  options: FilterOption[];
  multiple?: boolean;
  /**
   * Hands the typed text to the caller instead of filtering the options locally,
   * for filters whose options come from a server search.
   */
  onSearchChange?: (query: string) => void;
  searching?: boolean;
};

export type AddFilterMenuProps = {
  filters: FilterDefinition[];
  values: Record<string, string[]>;
  onChange: (filterId: string, values: string[]) => void;
  initialFilterId?: string | null;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  className?: string;
};

export function AddFilterMenu({
  filters,
  values,
  onChange,
  initialFilterId = null,
  open: controlledOpen,
  onOpenChange,
  className,
}: AddFilterMenuProps) {
  const t = useT();
  const [uncontrolledOpen, setUncontrolledOpen] = useState(false);
  const [activeId, setActiveId] = useState<string | null>(initialFilterId);
  const [search, setSearch] = useState("");
  const open = controlledOpen ?? uncontrolledOpen;
  const active = filters.find((filter) => filter.id === activeId) ?? null;
  const remoteSearch = active?.onSearchChange;

  const changeActive = (next: string | null) => {
    setActiveId(next);
    setSearch("");
    remoteSearch?.("");
  };

  const setOpen = (next: boolean) => {
    if (!next) changeActive(initialFilterId);
    if (controlledOpen === undefined) setUncontrolledOpen(next);
    onOpenChange?.(next);
  };

  const toggle = (filter: FilterDefinition, value: string) => {
    const current = values[filter.id] ?? [];
    if (!filter.multiple) {
      onChange(filter.id, current.includes(value) ? [] : [value]);
      setOpen(false);
      return;
    }
    onChange(
      filter.id,
      current.includes(value) ? current.filter((v) => v !== value) : [...current, value],
    );
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button variant="outline" className={cn("gap-1.5", className)}>
            <ListFilterIcon className="size-3.5" aria-hidden="true" />
            {t("Filter")}
          </Button>
        }
      />
      <PopoverContent align="start" className="w-64 p-0">
        <Command shouldFilter={!remoteSearch}>
          {active && (
            <button
              type="button"
              onClick={() => changeActive(null)}
              className="flex w-full items-center gap-1 border-b px-2 py-1.5 text-xs text-muted-foreground hover:text-foreground"
            >
              <ChevronLeftIcon className="size-3.5" aria-hidden="true" />
              {active.label}
            </button>
          )}
          <CommandInput
            value={search}
            onValueChange={(next) => {
              setSearch(next);
              remoteSearch?.(next);
            }}
            placeholder={active ? t("Filter {0}…", active.label) : t("Filter by…")}
          />
          <CommandList>
            <CommandEmpty>{active?.searching ? t("Searching…") : t("No matches")}</CommandEmpty>
            {active ? (
              <CommandGroup>
                {active.options.map((option) => {
                  const selected = (values[active.id] ?? []).includes(option.value);
                  return (
                    <CommandItem
                      key={option.value}
                      value={`${option.value} ${option.keywords ?? ""}`}
                      onSelect={() => toggle(active, option.value)}
                    >
                      {option.icon}
                      <span className="flex-1 truncate">{option.label}</span>
                      {selected && <CheckIcon className="size-3.5" aria-hidden="true" />}
                    </CommandItem>
                  );
                })}
              </CommandGroup>
            ) : (
              <CommandGroup>
                {filters.map((filter) => (
                  <CommandItem
                    key={filter.id}
                    value={filter.label}
                    onSelect={() => changeActive(filter.id)}
                  >
                    {filter.icon}
                    <span className="flex-1">{filter.label}</span>
                    {(values[filter.id]?.length ?? 0) > 0 && (
                      <span className="text-xs text-muted-foreground tabular-nums">
                        {values[filter.id]?.length}
                      </span>
                    )}
                  </CommandItem>
                ))}
              </CommandGroup>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
