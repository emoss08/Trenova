import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import type { FilterStateSchema } from "@/lib/schemas/table-configuration-schema";
import { Config, EnhancedColumnDef } from "@/types/data-table";
import { faBarsFilter } from "@fortawesome/pro-regular-svg-icons";
import { DataTableFilterContent } from "./data-table-filter-content";

interface DataTableFiltersProps {
  columns: EnhancedColumnDef<any>[];
  filterState: FilterStateSchema;
  onFilterChange: (state: FilterStateSchema) => void;
  config?: Config;
}

export function DataTableFilters({
  columns,
  filterState,
  onFilterChange,
  config,
}: DataTableFiltersProps) {
  if (!config?.showFilterUI) {
    return null;
  }

  return (
    <DataTableFiltersInner>
      <Popover>
        <PopoverTrigger asChild>
          <Button variant="outline" className="flex items-center gap-2">
            <Icon icon={faBarsFilter} className="size-4" />
            <span className="text-sm">Filter</span>
            {filterState.filters.length > 0 && (
              <Badge
                withDot={false}
                className="h-[18.24px] rounded-[3.2px] px-[5.12px] text-xs"
                variant="secondary"
              >
                {filterState.filters.length}
              </Badge>
            )}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-auto sm:min-w-[380px] p-0" align="start">
          <DataTableFilterContent
            filterState={filterState}
            columns={columns}
            onFilterChange={onFilterChange}
          />
        </PopoverContent>
      </Popover>
    </DataTableFiltersInner>
  );
}

function DataTableFiltersInner({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex flex-col items-center gap-2 w-full lg:w-auto">
      {children}
    </div>
  );
}
