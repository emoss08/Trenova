"use no memo";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { cn } from "@/lib/utils";
import type { DockAction } from "@/types/data-table";
import type { Table } from "@tanstack/react-table";
import { ChevronDownIcon, XIcon } from "lucide-react";
import { AnimatePresence, m } from "motion/react";
import { useCallback, useState } from "react";
import { Command, CommandGroup, CommandItem, CommandList } from "../ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "../ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "../ui/tooltip";

type DataTableDockProps<TData> = {
  table: Table<TData>;
  actions: DockAction<TData>[];
};

export function DataTableDock<TData>({
  table,
  actions,
}: DataTableDockProps<TData>) {
  const [loadingActions, setLoadingActions] = useState<Set<string>>(new Set());
  const [openSelectId, setOpenSelectId] = useState<string | null>(null);
  const selectedRows = table.getFilteredSelectedRowModel().rows;
  const selectedCount = selectedRows.length;

  const handleClearSelection = useCallback(() => {
    table.toggleAllRowsSelected(false);
  }, [table]);

  const getSelectedData = useCallback((): TData[] => {
    return selectedRows.map((row) => row.original);
  }, [selectedRows]);

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
        await result.finally(() => {
          setLoadingActions((prev) => {
            const next = new Set(prev);
            next.delete(action.id);
            return next;
          });
          if (action.clearSelectionOnSuccess) {
            handleClearSelection();
          }
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
          <div className="flex items-center gap-1 rounded-lg border border-background/20 bg-foreground px-2 py-1.5 shadow-lg">
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    variant="ghost"
                    className="h-7 cursor-pointer rounded-md border border-muted-foreground/40 text-muted-foreground shadow-none hover:bg-accent/20 dark:hover:bg-accent/20"
                    size="sm"
                    onClick={handleClearSelection}
                  >
                    <span className="text-sm font-medium text-background tabular-nums">
                      {selectedCount} selected
                    </span>{" "}
                    <XIcon className="size-3 text-background" />
                  </Button>
                }
              />
              <TooltipContent sideOffset={10}>Clear selection</TooltipContent>
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
                      onOpenChange={(open) =>
                        setOpenSelectId(open ? action.id : null)
                      }
                    >
                      <PopoverTrigger
                        render={
                          <Button
                            variant={
                              action.variant === "destructive"
                                ? "destructive"
                                : "ghost"
                            }
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
                              ? action.loadingLabel
                              : action.label}
                            <ChevronDownIcon className="size-3 opacity-50 transition-transform duration-200 group-data-[state=open]:rotate-180" />
                          </Button>
                        }
                      />
                      <PopoverContent
                        className="dark w-[180px] border-input p-0"
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
                                  onSelect={() =>
                                    handleSelectAction(action, option.value)
                                  }
                                  className="flex items-center gap-2 text-xs"
                                >
                                  {option.color && (
                                    <span
                                      className="block size-2 rounded-full"
                                      style={{ backgroundColor: option.color }}
                                    />
                                  )}
                                  <span>{option.label}</span>
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
                    variant={
                      action.variant === "destructive" ? "destructive" : "ghost"
                    }
                    size="sm"
                    disabled={isLoading}
                    className={cn(
                      "h-7 gap-1.5 px-2",
                      action.variant !== "destructive" &&
                        "text-background hover:bg-accent/20 hover:text-background dark:hover:bg-accent/20",
                    )}
                    onClick={() => handleSimpleActionClick(action)}
                  >
                    {isLoading ? (
                      <Spinner />
                    ) : (
                      ActionIcon && <ActionIcon className="size-4" />
                    )}
                    {isLoading && action.loadingLabel
                      ? action.loadingLabel
                      : action.label}
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
