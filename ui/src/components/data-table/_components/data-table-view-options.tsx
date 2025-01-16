import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Icon } from "@/components/ui/icons";

import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn, toSentenceCase } from "@/lib/utils";
import { useTableStore } from "@/stores/table-store";
import {
  DataTableCreateButtonProps,
  DataTableViewOptionsProps,
} from "@/types/data-table";
import { faPlus } from "@fortawesome/pro-solid-svg-icons";
import {
  CheckIcon,
  MixerHorizontalIcon,
  PlusIcon,
  UploadIcon,
} from "@radix-ui/react-icons";
import React, { useCallback } from "react";

export function DataTableCreateButton({
  name,
  isDisabled,
  onCreateClick,
}: DataTableCreateButtonProps) {
  const handleClick = useCallback(() => {
    // Handle create logic
    if (onCreateClick) {
      onCreateClick();
    } else {
      useTableStore.set("showCreateModal", true);
    }
  }, [onCreateClick]);

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="default" disabled={isDisabled}>
          <Icon icon={faPlus} className="size-4 text-background" />
          <span>New</span>
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" side="bottom" className="w-auto p-1">
        <div className="flex w-full flex-col gap-1">
          <Button
            variant="ghost"
            className="flex size-full flex-col items-start gap-0.5 text-left"
            onClick={handleClick}
          >
            <div className="flex items-center gap-2">
              <PlusIcon className="size-4" />
              <span>Add New {name}</span>
            </div>
            <div>
              <p className="text-xs font-normal text-muted-foreground">
                Create a new {name} from scratch
              </p>
            </div>
          </Button>
          <Button
            variant="ghost"
            className="flex size-full flex-col items-start gap-0.5 text-left"
          >
            <div className="flex items-center gap-2">
              <UploadIcon className="size-4" />
              <span>Import {name}s</span>
            </div>
            <div>
              <p className="text-xs font-normal text-muted-foreground">
                Import existing {name}s from a file
              </p>
            </div>
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}

export function DataTableViewOptions<TData>({
  table,
}: DataTableViewOptionsProps<TData>) {
  const [open, setOpen] = React.useState(false);

  return (
    <Popover open={open} onOpenChange={(open) => setOpen(open)}>
      <PopoverTrigger asChild>
        <Button
          variant="ghost"
          className="h-7"
          aria-label="Toggle columns"
          role="combobox"
        >
          <MixerHorizontalIcon className="size-3" />
          View
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" side="bottom" className="w-44 p-0">
        <Command>
          <CommandList>
            <CommandInput className="h-7" placeholder="Search columns..." />
            <CommandEmpty>No columns found.</CommandEmpty>
            <CommandGroup>
              {table
                .getAllColumns()
                .filter(
                  (column) =>
                    typeof column.accessorFn !== "undefined" &&
                    column.getCanHide(),
                )
                .map((column) => {
                  return (
                    <CommandItem
                      key={column.id}
                      onSelect={() =>
                        column.toggleVisibility(!column.getIsVisible())
                      }
                    >
                      <span className="truncate">
                        {toSentenceCase(column.id)}
                      </span>
                      <CheckIcon
                        className={cn(
                          "ml-auto size-4 shrink-0",
                          column.getIsVisible() ? "opacity-100" : "opacity-0",
                        )}
                      />
                    </CommandItem>
                  );
                })}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
