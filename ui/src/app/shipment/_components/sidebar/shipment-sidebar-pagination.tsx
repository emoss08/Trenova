import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { type ShipmentPaginationProps } from "@/types/shipment";
import {
    faChevronLeft,
    faChevronRight,
} from "@fortawesome/pro-regular-svg-icons";

export function ShipmentPagination({
  totalCount,
  page,
  pageSize,
  onPageChange,
  onPageSizeChange,
  pageSizeOptions,
  isLoading,
}: ShipmentPaginationProps) {
  const totalPages = Math.ceil(totalCount / pageSize);

  const start = (page - 1) * pageSize + 1;
  const end = Math.min(page * pageSize, totalCount);

  return totalCount > 0 ? (
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
  ) : null;
}
