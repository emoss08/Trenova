/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

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
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { CheckIcon } from "@radix-ui/react-icons";
import { ChevronDown } from "lucide-react";
import { useState } from "react";

type FilterOption = {
  value: string | boolean | number;
  label: string;
};

type FilterOptions = {
  id: string;
  title: string;
  options: FilterOption[];
  loading?: boolean;
};

function Filter({ title, options, loading }: FilterOptions) {
  const [selectedValues, setSelectedValues] = useState<
    (string | boolean | number)[]
  >([]);

  return (
    <Popover>
      <PopoverTrigger asChild>
        <div className="flex items-center space-x-0.5 text-sm text-muted-foreground hover:cursor-pointer hover:text-foreground">
          <span className="truncate">{title}</span>
          <ChevronDown className="size-4" />
        </div>
      </PopoverTrigger>
      <PopoverContent className="w-[200px] p-0" align="start">
        <Command>
          <CommandInput placeholder={title} />
          <CommandList>
            {loading ? (
              <CommandItem>
                <Skeleton className="h-6 w-full" />
              </CommandItem>
            ) : (
              <>
                {options.length === 0 && (
                  <CommandEmpty>No Results Found.</CommandEmpty>
                )}
                <CommandGroup>
                  {options.map((option) => {
                    const isSelected = selectedValues.includes(option.value);
                    return (
                      <CommandItem
                        key={option.value.toString()}
                        onSelect={() => {
                          const newValueSet = new Set(selectedValues);
                          if (newValueSet.has(option.value)) {
                            newValueSet.delete(option.value);
                          } else {
                            newValueSet.add(option.value);
                          }
                          setSelectedValues(Array.from(newValueSet));
                        }}
                      >
                        <div
                          className={cn(
                            "mr-2 flex h-4 w-4 items-center justify-center rounded-sm border border-primary",
                            isSelected
                              ? "bg-primary text-primary-foreground"
                              : "opacity-50 [&_svg]:invisible",
                          )}
                        >
                          <CheckIcon className={cn("h-4 w-4")} />
                        </div>
                        <span>{option.label}</span>
                      </CommandItem>
                    );
                  })}
                </CommandGroup>
              </>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

export function WorkerSortOptions({
  sortOptions,
}: {
  sortOptions: FilterOptions[];
}) {
  return (
    <div className="mt-2 flex items-center justify-between">
      {sortOptions.map((sortOptions) => (
        <Filter key={sortOptions.id} {...sortOptions} />
      ))}
    </div>
  );
}
