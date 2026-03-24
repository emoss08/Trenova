"use no memo";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import type { Table } from "@tanstack/react-table";
import { CheckIcon, Columns3Icon } from "lucide-react";
import { useState } from "react";

type DataTableViewOptionsProps<TData> = {
  table: Table<TData>;
};

export function DataTableViewOptions<TData>({
  table,
}: DataTableViewOptionsProps<TData>) {
  const [open, setOpen] = useState(false);

  const columns = table
    .getAllColumns()
    .filter(
      (column) =>
        typeof column.accessorFn !== "undefined" && column.getCanHide(),
    );

  if (columns.length === 0) {
    return null;
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button variant="outline" size="sm" className="h-8">
            <Columns3Icon className="size-4" />
            <span className="hidden pt-0.5 lg:inline">Columns</span>
          </Button>
        }
      />
      <PopoverContent className="dark w-48 p-0" align="end">
        <Command>
          <CommandInput className="h-8" placeholder="Search columns..." />
          <CommandList>
            <CommandEmpty>No columns found.</CommandEmpty>
            <CommandGroup>
              {columns.map((column) => {
                const label = column.columnDef.meta?.label || column.id;
                const isVisible = column.getIsVisible();
                return (
                  <CommandItem
                    key={column.id}
                    value={label}
                    onSelect={() => column.toggleVisibility(!isVisible)}
                  >
                    <CheckIcon
                      className={cn(
                        "size-3.5",
                        isVisible ? "opacity-100" : "opacity-0",
                      )}
                    />
                    {label}
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

export default DataTableViewOptions;
