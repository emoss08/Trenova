import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { cn } from "@trenova/shared/lib/utils";
import type { DockAction } from "@trenova/shared/types/data-table";
import { ChevronDownIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { AnimatePresence, m } from "motion/react";
import { useCallback, useState } from "react";
import {
  Command,
  CommandGroup,
  CommandItem,
  CommandList,
} from "@trenova/shared/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";

/** A figure the dock shows beside the count, already formatted. */
export type SelectionDockTotal = {
  id: string;
  label: string;
  text: string;
};

type SelectionDockProps<TData> = {
  selectedCount: number;
  totals?: readonly SelectionDockTotal[];
  actions: DockAction<TData>[];
  getSelected: () => TData[];
  onClearSelection: () => void;
};

/**
 * The floating bar that appears over a list once rows are selected: how many,
 * a way to clear them, and what can be done to all of them at once. Data
 * tables and the billing work lists share it, so a bulk action looks and
 * behaves the same wherever it is offered.
 */
export function SelectionDock<TData>({
  selectedCount,
  totals,
  actions,
  getSelected,
  onClearSelection,
}: SelectionDockProps<TData>) {
  const t = useT();

  const [loadingActions, setLoadingActions] = useState<Set<string>>(new Set());
  const [openSelectId, setOpenSelectId] = useState<string | null>(null);
  const handleClearSelection = onClearSelection;
  const getSelectedData = getSelected;
  const selectionTotals = totals ?? [];

  const handleSimpleActionClick = useCallback(
    async (action: DockAction<TData>) => {
      if (action.type === "select") return;

      const data = getSelectedData();
      const result = action.onClick(data);

      if (result instanceof Promise) {
        setLoadingActions((prev) => new Set(prev).add(action.id));
        await result
          .then(() => {
            if (action.clearSelectionOnSuccess) {
              handleClearSelection();
            }
          })
          .catch(() => undefined)
          .finally(() => {
            setLoadingActions((prev) => {
              const next = new Set(prev);
              next.delete(action.id);
              return next;
            });
          });
      } else if (action.clearSelectionOnSuccess) {
        handleClearSelection();
      }
    },
    [getSelectedData, handleClearSelection],
  );

  const handleSelectAction = useCallback(
    async (action: DockAction<TData>, value: string) => {
      if (action.type !== "select") return;

      const data = getSelectedData();
      setOpenSelectId(null);

      const result = action.onSelect(data, value);

      if (result instanceof Promise) {
        setLoadingActions((prev) => new Set(prev).add(action.id));
        await result
          .then(() => {
            if (action.clearSelectionOnSuccess) {
              handleClearSelection();
            }
          })
          .catch(() => undefined)
          .finally(() => {
            setLoadingActions((prev) => {
              const next = new Set(prev);
              next.delete(action.id);
              return next;
            });
          });
      } else if (action.clearSelectionOnSuccess) {
        handleClearSelection();
      }
    },
    [getSelectedData, handleClearSelection],
  );

  return (
    <AnimatePresence>
      {selectedCount > 0 && (
        <m.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: 20 }}
          transition={{ duration: 0.2, ease: "easeOut" }}
          className="fixed bottom-6 left-1/2 z-50 -translate-x-1/2"
        >
          <div className="border-background/20 bg-foreground flex items-center gap-1 rounded-lg border px-2 py-1.5">
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    variant="ghost"
                    className="border-muted-foreground/40 text-muted-foreground hover:bg-accent/20 dark:hover:bg-accent/20 h-7 cursor-pointer rounded-md border shadow-none"
                    size="sm"
                    onClick={handleClearSelection}
                  >
                    <span className="text-background text-sm font-medium tabular-nums">
                      {t("{0} selected", selectedCount)}
                    </span>
                    {selectionTotals.map((total) => (
                      <span key={total.id} className="text-background/70 text-xs tabular-nums">
                        {t("{0}: {1}", total.label, total.text)}
                      </span>
                    ))}{" "}
                    <XCloseIcon className="text-background size-3" />
                  </Button>
                }
              />
              <TooltipContent sideOffset={10}>{t("Clear selection")}</TooltipContent>
            </Tooltip>
            <div className="flex items-center gap-1 pl-1">
              {actions.map((action) => {
                const isLoading = loadingActions.has(action.id);
                const ActionIcon = action.icon;

                if (action.type === "select") {
                  return (
                    <Popover
                      key={action.id}
                      open={openSelectId === action.id}
                      onOpenChange={(open) => setOpenSelectId(open ? action.id : null)}
                    >
                      <PopoverTrigger
                        render={
                          <Button
                            variant={action.variant === "destructive" ? "destructive" : "ghost"}
                            size="sm"
                            disabled={isLoading}
                            className={cn(
                              "group h-7 gap-1.5 px-2",
                              action.variant !== "destructive" &&
                                "text-background hover:bg-accent/20 hover:text-background dark:hover:bg-accent/20",
                            )}
                          >
                            {isLoading ? (
                              <Spinner />
                            ) : (
                              ActionIcon && <ActionIcon className="size-4" />
                            )}
                            {isLoading && action.loadingLabel
                              ? t(action.loadingLabel)
                              : t(action.label)}
                            <ChevronDownIcon className="size-3 opacity-50 transition-transform duration-200 group-data-[state=open]:rotate-180" />
                          </Button>
                        }
                      />
                      <PopoverContent
                        className="border-input w-[180px] p-0"
                        align="start"
                        sideOffset={10}
                      >
                        <Command>
                          <CommandList>
                            <CommandGroup>
                              {action.options.map((option) => (
                                <CommandItem
                                  key={option.value}
                                  value={option.value}
                                  onSelect={() => handleSelectAction(action, option.value)}
                                  className="flex items-center gap-2 text-xs"
                                >
                                  {option.color && (
                                    <span
                                      className="block size-2 rounded-full"
                                      style={{ backgroundColor: option.color }}
                                    />
                                  )}
                                  <span>{t(option.label)}</span>
                                </CommandItem>
                              ))}
                            </CommandGroup>
                          </CommandList>
                        </Command>
                      </PopoverContent>
                    </Popover>
                  );
                }

                return (
                  <Button
                    key={action.id}
                    variant={action.variant === "destructive" ? "destructive" : "ghost"}
                    size="sm"
                    disabled={isLoading}
                    className={cn(
                      "h-7 gap-1.5 px-2",
                      action.variant !== "destructive" &&
                        "text-background hover:bg-accent/20 hover:text-background dark:hover:bg-accent/20",
                    )}
                    onClick={() => handleSimpleActionClick(action)}
                  >
                    {isLoading ? <Spinner /> : ActionIcon && <ActionIcon className="size-4" />}
                    {isLoading && action.loadingLabel ? t(action.loadingLabel) : t(action.label)}
                  </Button>
                );
              })}
            </div>
          </div>
        </m.div>
      )}
    </AnimatePresence>
  );
}
