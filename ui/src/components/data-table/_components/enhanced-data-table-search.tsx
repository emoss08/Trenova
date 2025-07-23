/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { Input } from "@/components/ui/input";
import { createDebouncedSearch } from "@/lib/enhanced-data-table-utils";
import type { FilterStateSchema } from "@/lib/schemas/table-configuration-schema";
import { faSearch } from "@fortawesome/pro-regular-svg-icons";
import { X } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

interface EnhancedDataTableSearchProps {
  filterState: FilterStateSchema;
  onFilterChange: (state: FilterStateSchema) => void;
  placeholder?: string;
  debounceMs?: number;
  className?: string;
}

export function EnhancedDataTableSearch({
  filterState,
  onFilterChange,
  placeholder = "Search...",
  debounceMs = 300,
}: EnhancedDataTableSearchProps) {
  const [searchValue, setSearchValue] = useState(
    filterState.globalSearch || "",
  );

  // Create debounced search function
  const debouncedSearch = createDebouncedSearch(
    useCallback(
      (query: string) => {
        onFilterChange({
          ...filterState,
          globalSearch: query,
        });
      },
      [filterState, onFilterChange],
    ),
    debounceMs,
  );

  // Handle search input changes
  const handleSearchChange = (value: string) => {
    setSearchValue(value);
    debouncedSearch(value);
  };

  // Clear search
  const handleClearSearch = () => {
    setSearchValue("");
    onFilterChange({
      ...filterState,
      globalSearch: "",
    });
  };

  // Sync with external filter state changes
  useEffect(() => {
    setSearchValue(filterState.globalSearch || "");
  }, [filterState.globalSearch]);

  return (
    <div className="flex items-center max-w-[200px]">
      <Input
        type="text"
        placeholder={placeholder}
        value={searchValue}
        onChange={(e) => handleSearchChange(e.target.value)}
        icon={<Icon icon={faSearch} className="size-3 text-muted-foreground" />}
      />
      {searchValue && (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={handleClearSearch}
          className="absolute right-1 top-1/2 transform -translate-y-1/2 h-6 w-6 p-0 hover:bg-muted"
        >
          <X className="size-3" />
          <span className="sr-only">Clear search</span>
        </Button>
      )}
    </div>
  );
}
