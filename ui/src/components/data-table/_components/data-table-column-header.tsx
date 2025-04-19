import { SelectIcon } from "@radix-ui/react-select";
import { type Column } from "@tanstack/react-table";

import { Icon } from "@/components/ui/icons";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from "@/components/ui/select";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import {
  faArrowDown,
  faArrowUp,
  faArrowUpArrowDown,
  faCircleInfo,
} from "@fortawesome/pro-regular-svg-icons";
import { ArrowDownIcon, ArrowUpIcon, EyeNoneIcon } from "@radix-ui/react-icons";

interface DataTableColumnHeaderProps<TData, TValue>
  extends React.HTMLAttributes<HTMLDivElement> {
  column: Column<TData, TValue>;
  title: string;
}

export function DataTableColumnHeader<TData, TValue>({
  column,
  title,
  className,
}: DataTableColumnHeaderProps<TData, TValue>) {
  if (!column.getCanSort() && !column.getCanHide()) {
    return <div className={cn(className)}>{title}</div>;
  }

  const ascValue = `${column.id}-asc`;
  const descValue = `${column.id}-desc`;
  const hideValue = `${column.id}-hide`;

  return (
    <div className={cn("flex items-center gap-2", className)}>
      <Select
        value={
          column.getIsSorted() === "desc"
            ? descValue
            : column.getIsSorted() === "asc"
              ? ascValue
              : undefined
        }
        onValueChange={(value) => {
          if (value === ascValue) column.toggleSorting(false);
          else if (value === descValue) column.toggleSorting(true);
          else if (value === hideValue) column.toggleVisibility(false);
        }}
      >
        <SelectTrigger
          aria-label={
            column.getIsSorted() === "desc"
              ? "Sorted descending. Click to sort ascending."
              : column.getIsSorted() === "asc"
                ? "Sorted ascending. Click to sort descending."
                : "Not sorted. Click to sort ascending."
          }
          className="inline-flex items-center justify-between -ml-3 h-8 w-fit border-transparent bg-transparent text-xs hover:bg-accent hover:text-accent-foreground data-[state=open]:bg-accent data-[state=open]:text-accent-foreground [&>svg:last-child]:hidden"
        >
          <div className="inline-flex items-center gap-1">
            {title}
            <SelectIcon asChild className="inline-flex items-center">
              {column.getCanSort() && column.getIsSorted() === "desc" ? (
                <Icon
                  icon={faArrowDown}
                  className="size-3 translate-y-[0.5px]"
                  aria-hidden="true"
                />
              ) : column.getIsSorted() === "asc" ? (
                <Icon
                  icon={faArrowUp}
                  className="size-3 translate-y-[0.5px]"
                  aria-hidden="true"
                />
              ) : (
                <Icon
                  icon={faArrowUpArrowDown}
                  className="size-3 translate-y-[0.5px]"
                  aria-hidden="true"
                />
              )}
            </SelectIcon>
          </div>
        </SelectTrigger>
        <SelectContent align="start" className="min-w-[8rem]">
          {column.getCanSort() && (
            <>
              <SelectItem value={ascValue}>
                <span className="inline-flex items-center gap-2">
                  <ArrowUpIcon
                    className="size-3.5 text-muted-foreground/70 translate-y-[0.5px]"
                    aria-hidden="true"
                  />
                  <span>Asc</span>
                </span>
              </SelectItem>
              <SelectItem value={descValue}>
                <span className="inline-flex items-center gap-2">
                  <ArrowDownIcon
                    className="size-3.5 text-muted-foreground/70 translate-y-[0.5px]"
                    aria-hidden="true"
                  />
                  <span>Desc</span>
                </span>
              </SelectItem>
            </>
          )}
          {column.getCanHide() && (
            <SelectItem value={hideValue}>
              <span className="inline-flex items-center gap-2">
                <EyeNoneIcon
                  className="size-3.5 text-muted-foreground/70 translate-y-[0.5px]"
                  aria-hidden="true"
                />
                <span>Hide</span>
              </span>
            </SelectItem>
          )}
        </SelectContent>
      </Select>
    </div>
  );
}

type DataTableColumnHeaderWithTooltipProps<TData, TValue> =
  DataTableColumnHeaderProps<TData, TValue> & {
    title: string;
    tooltipContent: string;
    startContent?: React.ReactNode;
  };

export function DataTableColumnHeaderWithTooltip<TData, TValue>({
  title,
  tooltipContent,
  column,
  className,
  startContent,
}: DataTableColumnHeaderWithTooltipProps<TData, TValue>) {
  if (!column.getCanSort() && !column.getCanHide()) {
    return <div className={cn(className)}>{title}</div>;
  }

  return (
    <div className={cn("flex items-center gap-2", className)}>
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="inline-flex items-center justify-center cursor-normal -ml-3 h-8 px-3 py-2 text-sm rounded-md w-fit border-transparent bg-transparent text-xs hover:bg-accent hover:text-accent-foreground data-[state=open]:bg-accent data-[state=open]:text-accent-foreground [&>svg:last-child]:hidden">
            <div className="inline-flex items-center gap-1">
              {startContent}
              {title}
              <Icon
                icon={faCircleInfo}
                className="inline-flex items-center size-3"
                aria-hidden="true"
              />
            </div>
          </div>
        </TooltipTrigger>
        <TooltipContent>{tooltipContent}</TooltipContent>
      </Tooltip>
    </div>
  );
}
