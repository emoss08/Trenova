import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { Input } from "@/components/ui/input";
import type { FilterStateSchema } from "@/lib/schemas/table-configuration-schema";
import { faSearch } from "@fortawesome/pro-regular-svg-icons";
import { X } from "lucide-react";
import { useCallback, useEffectEvent, useRef, useState } from "react";

interface EnhancedDataTableSearchProps {
  filterState: FilterStateSchema;
  onFilterChange: (state: FilterStateSchema) => void;
  placeholder?: string;
  className?: string;
}

export function DataTableSearch({
  filterState,
  onFilterChange,
  placeholder = "Search...",
}: EnhancedDataTableSearchProps) {
  const [localValue, setLocalValue] = useState(filterState.globalSearch || "");
  const isTypingRef = useRef(false);

  useEffectEvent(() => {
    if (!isTypingRef.current) {
      setLocalValue(filterState.globalSearch || "");
    }
  });

  const handleSearchChange = useCallback(
    (value: string) => {
      setLocalValue(value);
      isTypingRef.current = true;

      onFilterChange({
        ...filterState,
        globalSearch: value,
      });

      setTimeout(() => {
        isTypingRef.current = false;
      }, 500);
    },
    [filterState, onFilterChange],
  );

  const handleClearSearch = useCallback(() => {
    setLocalValue("");
    isTypingRef.current = false;
    onFilterChange({
      ...filterState,
      globalSearch: "",
    });
  }, [filterState, onFilterChange]);

  return (
    <div className="flex items-center relative">
      <Input
        id="table-search-input"
        type="text"
        placeholder={placeholder}
        value={localValue}
        className="w-full truncate pr-8"
        onChange={(e) => handleSearchChange(e.target.value)}
        icon={<Icon icon={faSearch} className="size-3 text-muted-foreground" />}
      />
      {localValue && (
        <Button
          type="button"
          variant="ghost"
          size="icon"
          onClick={handleClearSearch}
          className="absolute right-1 top-1/2 transform -translate-y-1/2 h-6 w-6 p-0"
        >
          <X className="size-4" />
          <span className="sr-only">Clear search</span>
        </Button>
      )}
    </div>
  );
}
