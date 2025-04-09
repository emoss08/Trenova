import { Checkbox } from "@/components/ui/checkbox";
import { InternalLink } from "@/components/ui/link";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { generateDateOnlyString, toDate } from "@/lib/date";
import { BaseModel } from "@/types/common";
import { ColumnDef, ColumnHelper } from "@tanstack/react-table";
import { v4 } from "uuid";
import { DataTableColumnHeader } from "./data-table-column-header";

type EntityRefConfig<TEntity, TParent> = {
  basePath: string;
  getId: (entity: TEntity) => string | undefined;
  getDisplayText: (entity: TEntity) => string;
  getHeaderText?: string;
  getSecondaryInfo?: (
    entity: TEntity,
    parent: TParent,
  ) => {
    label?: string;
    entity: TEntity;
    displayText: string;
    clickable?: boolean;
  } | null;
  className?: string;
  color?: {
    getColor: (entity: TEntity) => string | undefined;
  };
};

type NestedEntityRefConfig<TEntity, TParent> = EntityRefConfig<
  TEntity,
  TParent
> & {
  getEntity: (parent: TParent) => TEntity | null | undefined;
  columnId?: string;
};

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
      size: 50,
      enableSorting: false,
      enableHiding: false,
    }),
    createdAt: createdAtColumn(columnHelper) as ColumnDef<T>,
  };
}

function createdAtColumn<T extends Record<string, unknown>>(
  columnHelper: ColumnHelper<T>,
) {
  return columnHelper.accessor(
    (row) => (row.original as unknown as BaseModel).createdAt,
    {
      id: "createdAt",
      header: "Created At",
      cell: ({ row }) => {
        const { createdAt } = row.original;
        const date = toDate(createdAt as number);
        if (!date) return <p>-</p>;

        return <p>{generateDateOnlyString(date)}</p>;
      },
    },
  );
}

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
        column={column as any}
        title={config.getHeaderText ?? ""}
      />
    ),
    cell: ({ getValue, row }) => {
      const entity = getValue();

      if (!entity) {
        return <p className="text-muted-foreground">-</p>;
      }

      const id = config.getId(entity);
      const displayText = config.getDisplayText(entity);
      const secondaryInfo = config.getSecondaryInfo?.(entity, row.original);
      const color = config.color?.getColor(entity);

      // clickable should default to true unless otherwise specified
      const clickable = secondaryInfo?.clickable ?? true;

      return (
        <div className="flex flex-col gap-0.5">
          <Tooltip delayDuration={300}>
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

          {secondaryInfo && (
            <div className="flex items-center gap-1 text-muted-foreground text-2xs">
              {secondaryInfo.label && <span>{secondaryInfo.label}:</span>}
              <Tooltip delayDuration={300}>
                <TooltipTrigger asChild>
                  {clickable ? (
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
                  ) : (
                    <p>{secondaryInfo.displayText}</p>
                  )}
                </TooltipTrigger>
                <TooltipContent>
                  <p>Click to view {secondaryInfo.displayText}</p>
                </TooltipContent>
              </Tooltip>
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
        column={column as any}
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

export function createNestedEntityRefColumn<
  T extends Record<string, any>,
  TValue,
>(
  columnHelper: ColumnHelper<T>,
  config: NestedEntityRefConfig<TValue, T>,
): ColumnDef<T> {
  return columnHelper.accessor((row) => config.getEntity(row), {
    id: config.columnId ?? v4(),
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column as any}
        title={config.getHeaderText ?? ""}
      />
    ),
    cell: ({ getValue, row }) => {
      const entity = getValue();

      if (!entity) {
        return <p className="text-muted-foreground">-</p>;
      }

      const id = config.getId(entity);
      const displayText = config.getDisplayText(entity);
      const secondaryInfo = config.getSecondaryInfo?.(entity, row.original);
      const color = config.color?.getColor(entity);

      // clickable should default to true unless otherwise specified
      const clickable = secondaryInfo?.clickable ?? true;

      return (
        <div className="flex flex-col gap-0.5">
          <Tooltip delayDuration={300}>
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

          {secondaryInfo && (
            <div className="flex items-center gap-1 text-muted-foreground text-2xs">
              {secondaryInfo.label && <span>{secondaryInfo.label}:</span>}
              {clickable ? (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <InternalLink
                      to={{
                        pathname: config.basePath,

                        search: `?entityId=${config.getId(
                          secondaryInfo.entity,
                        )}&modal=edit`,
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
              ) : (
                <p>{secondaryInfo.displayText}</p>
              )}
            </div>
          )}
        </div>
      );
    },
  }) as ColumnDef<T>;
}
