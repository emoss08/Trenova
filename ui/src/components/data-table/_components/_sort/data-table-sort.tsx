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
import { faArrowUpArrowDown } from "@fortawesome/pro-regular-svg-icons";
import { DataTableSortContent } from "./data-table-sort-content";

interface DataTableSortProps {
  columns: EnhancedColumnDef<any>[];
  sortState: FilterStateSchema["sort"];
  onSortChange: (sort: FilterStateSchema["sort"]) => void;
  config?: Config;
}

export function DataTableSort({
  columns,
  sortState,
  onSortChange,
  config,
}: DataTableSortProps) {
  if (!config?.showSortUI) {
    return null;
  }

  return (
    <DataTableSortInner>
      <Popover>
        <PopoverTrigger asChild>
          <Button variant="outline" className="flex items-center gap-2">
            <Icon icon={faArrowUpArrowDown} className="size-4" />
            <span className="text-sm">Sort</span>
            {sortState.length > 0 && (
              <Badge
                withDot={false}
                className="h-[18.24px] rounded-[3.2px] px-[5.12px] text-xs"
                variant="secondary"
              >
                {sortState.length}
              </Badge>
            )}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-auto sm:min-w-[380px] p-0" align="start">
          <DataTableSortContent
            columns={columns}
            sortState={sortState}
            onSortChange={onSortChange}
            config={config}
          />
        </PopoverContent>
      </Popover>
    </DataTableSortInner>
  );
}

function DataTableSortInner({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex flex-col items-center gap-2 w-full lg:w-auto">
      {children}
    </div>
  );
}
