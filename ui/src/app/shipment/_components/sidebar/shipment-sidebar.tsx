import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { statusChoices } from "@/lib/choices";
import { ShipmentFilterSchema } from "@/lib/schemas/shipment-filter-schema";
import { type Shipment as ShipmentResponse } from "@/types/shipment";
import {
  faChevronLeft,
  faChevronRight,
  faFilter,
  faSearch,
} from "@fortawesome/pro-regular-svg-icons";
import { useFormContext } from "react-hook-form";
import { ShipmentCard } from "./shipment-card";
import { FilterOptions } from "./shipment-filter-options";

type ShipmentSidebarProps = {
  shipments: ShipmentResponse[];
  totalCount: number;
  page: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
  pageSizeOptions: readonly number[];
  isLoading: boolean;
};

export function ShipmentSidebar({
  shipments,
  totalCount,
  page,
  pageSize,
  onPageChange,
  onPageSizeChange,
  pageSizeOptions,
  isLoading,
}: ShipmentSidebarProps) {
  const { control } = useFormContext<ShipmentFilterSchema>();
  const totalPages = Math.ceil(totalCount / pageSize);

  const start = (page - 1) * pageSize + 1;
  const end = Math.min(page * pageSize, totalCount);

  return (
    <div className="flex flex-col h-full bg-sidebar rounded-md border border-sidebar-border">
      {/* Header section with filters */}
      <div className="flex-none p-2 space-y-2">
        <FilterOptions />
        <div className="flex flex-row gap-2 justify-start">
          <InputField
            control={control}
            name="search"
            placeholder="Search"
            className="h-7 w-[250px]"
            icon={
              <Icon
                icon={faSearch}
                className="size-3.5 text-muted-foreground"
              />
            }
          />
          <SelectField
            control={control}
            name="status"
            placeholder="Status"
            className="h-7 w-30"
            isClearable
            options={statusChoices}
          />
          <Button
            variant="outline"
            size="icon"
            className="border-muted-foreground/20 bg-muted border"
          >
            <Icon icon={faFilter} className="size-3.5" />
          </Button>
        </div>
      </div>

      {/* Scrollable shipments list with calculated height */}
      <div className="flex-1 min-h-0">
        <ScrollArea className="h-full">
          <div className="p-2 space-y-2">
            {shipments.map((shipment) => (
              <ShipmentCard key={shipment.id} shipment={shipment} />
            ))}
          </div>
        </ScrollArea>
      </div>

      {/* Fixed bottom section */}
      <div className="flex-none p-2 border-t border-sidebar-border bg-sidebar space-y-2">
        <div className="flex items-center justify-between">
          <Select
            value={pageSize.toString()}
            onValueChange={(value) => onPageSizeChange(Number(value))}
          >
            <SelectTrigger className="h-7 max-w-[100px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {pageSizeOptions.map((size) => (
                <SelectItem key={size} value={size.toString()}>
                  {size} / page
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <div className="flex items-center justify-center gap-2">
            <Button
              variant="outline"
              size="icon"
              className="h-7 w-7"
              disabled={page <= 1 || isLoading}
              onClick={() => onPageChange(page - 1)}
            >
              <Icon icon={faChevronLeft} className="size-3" />
            </Button>
            <div className="text-sm">
              Page {page} of {totalPages}
            </div>
            <Button
              variant="outline"
              size="icon"
              className="h-7 w-7"
              disabled={page >= totalPages || isLoading}
              onClick={() => onPageChange(page + 1)}
            >
              <Icon icon={faChevronRight} className="size-3" />
            </Button>
          </div>
        </div>
        <p className="flex items-center justify-center text-2xs text-muted-foreground">
          {totalCount > 0
            ? `Showing ${start}-${end} of ${totalCount} shipments`
            : "No shipments found"}
        </p>
      </div>
    </div>
  );
}
