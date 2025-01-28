import { Checkbox } from "@/components/ui/checkbox";
import { InternalLink } from "@/components/ui/link";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { ColumnDef, ColumnHelper } from "@tanstack/react-table";
import { DataTableColumnHeader } from "./data-table-column-header";

export function createCommonColumns<T extends Record<string, unknown>>(
  columnHelper: ColumnHelper<T>,
) {
  return {
    selection: columnHelper.display({
      id: "select",
      header: ({ table }) => {
        return (
          <Checkbox
            checked={
              table.getIsAllPageRowsSelected() ||
              (table.getIsSomePageRowsSelected() && "indeterminate")
            }
            onCheckedChange={(checked) =>
              table.toggleAllPageRowsSelected(!!checked)
            }
            aria-label="Select all"
          />
        );
      },
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(checked) => row.toggleSelected(!!checked)}
          aria-label="Select row"
        />
      ),
      enableSorting: false,
      enableHiding: false,
    }),
  };
}

type EntityRefConfig<TEntity, TParent> = {
  basePath: string;
  getId: (entity: TEntity) => string | undefined;
  getDisplayText: (entity: TEntity) => string;
  getHeaderText?: string;
  getSecondaryInfo?: (
    entity: TEntity,
    parent: TParent,
  ) => {
    label: string;
    entity: TEntity;
    displayText: string;
  } | null;
  className?: string;
  color?: {
    getColor: (entity: TEntity) => string | undefined;
  };
};

export function createEntityRefColumn<
  T extends Record<string, any>,
  K extends keyof T,
  TValue = T[K],
>(
  columnHelper: ColumnHelper<T>,
  accessorKey: K,
  config: EntityRefConfig<NonNullable<TValue>, T>,
): ColumnDef<T> {
  return columnHelper.accessor((row) => row[accessorKey], {
    id: accessorKey as string,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={config.getHeaderText ?? ""}
      />
    ),
    cell: ({ getValue, row }) => {
      const entity = getValue();

      if (!entity) {
        return (
          <p className="text-muted-foreground">
            No {config.basePath.split("/").pop()}
          </p>
        );
      }

      const id = config.getId(entity);
      const displayText = config.getDisplayText(entity);
      const secondaryInfo = config.getSecondaryInfo?.(entity, row.original);
      const color = config.color?.getColor(entity);

      return (
        <div className="flex flex-col gap-0.5">
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <InternalLink
                  to={{
                    pathname: config.basePath,
                    search: `?entityId=${id}&modal=edit`,
                  }}
                  state={{
                    isNavigatingToModal: true,
                  }}
                  className={config.className}
                  replace
                  preventScrollReset
                >
                  {color ? (
                    <div className="flex items-center gap-x-1.5 text-sm font-normal text-foreground underline hover:text-foreground/70">
                      <div
                        className="size-2 rounded-full"
                        style={{
                          backgroundColor: color,
                        }}
                      />
                      <p>{displayText}</p>
                    </div>
                  ) : (
                    <span className="text-sm font-normal underline hover:text-foreground/70">
                      {displayText}
                    </span>
                  )}
                </InternalLink>
              </TooltipTrigger>
              <TooltipContent>
                <p>Click to view {displayText}</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>

          {secondaryInfo && (
            <div className="flex items-center gap-1 text-muted-foreground text-2xs">
              <span>{secondaryInfo.label}:</span>
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <InternalLink
                      to={{
                        pathname: config.basePath,
                        search: `?entityId=${config.getId(secondaryInfo.entity)}&modal=edit`,
                      }}
                      state={{
                        isNavigatingToModal: true,
                      }}
                      className="text-2xs text-muted-foreground underline hover:text-muted-foreground/70"
                      replace
                      preventScrollReset
                      viewTransition
                    >
                      {secondaryInfo.displayText}
                    </InternalLink>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>Click to view {secondaryInfo.displayText}</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </div>
          )}
        </div>
      );
    },
  }) as ColumnDef<T>;
}

type EntityColumnConfig<T extends Record<string, any>, K extends keyof T> = {
  accessorKey: K;
  getHeaderText?: string;
  getId: (entity: T) => string | undefined;
  getDisplayText: (entity: T) => string;
  className?: string;
  getColor?: (entity: T) => string | undefined;
};

export function createEntityColumn<T extends Record<string, any>>(
  columnHelper: ColumnHelper<T>,
  accessorKey: keyof T,
  config: EntityColumnConfig<T, keyof T>,
): ColumnDef<T> {
  return columnHelper.accessor((row) => row[accessorKey], {
    id: accessorKey as string,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={config.getHeaderText ?? ""}
      />
    ),
    cell: ({ row }) => {
      const entity = row.original;

      if (!entity) {
        return <p>-</p>;
      }

      const id = config.getId(row.original);
      const displayText = config.getDisplayText(row.original);
      const color = config.getColor?.(row.original);

      return (
        <InternalLink
          to={{
            search: `?entityId=${id}&modal=edit`,
          }}
          state={{
            isNavigatingToModal: true,
          }}
          className={config.className}
          replace
          preventScrollReset
        >
          {color ? (
            <div className="flex items-center gap-x-1.5 text-sm font-normal text-foreground hover:text-foreground/70 w-fit underline">
              <div
                className="size-2 rounded-full"
                style={{
                  backgroundColor: color,
                }}
              />
              <p>{displayText}</p>
            </div>
          ) : (
            <span className="text-sm font-normal underline text-foreground hover:text-foreground/70">
              {displayText}
            </span>
          )}
        </InternalLink>
      );
    },
  }) as ColumnDef<T>;
}
