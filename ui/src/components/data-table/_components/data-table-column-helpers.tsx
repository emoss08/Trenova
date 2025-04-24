/* eslint-disable react-refresh/only-export-components */
/* eslint-disable react/display-name */
import { Checkbox } from "@/components/ui/checkbox";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { generateDateOnlyString, toDate } from "@/lib/date";
import { BaseModel } from "@/types/common";
import { ColumnDef, ColumnHelper } from "@tanstack/react-table";
import { parseAsString, useQueryState } from "nuqs";
import { memo, useCallback, useTransition } from "react";
import { v4 } from "uuid";
import {
  EntityColumnConfig,
  EntityRefConfig,
  NestedEntityRefConfig,
} from "./data-table-column-types";

// Entity parameter definitions - same as in data-table.tsx
const entityParams = {
  entityId: parseAsString,
  modal: parseAsString,
};

// Memoized EntityRefLink component to avoid re-renders
const EntityRefLink = memo(
  ({
    id,
    displayText,
    className,
    color,
  }: {
    id: string | undefined;
    displayText: string;
    className?: string;
    color?: string;
  }) => {
    const [, startTransition] = useTransition();

    // Use the nuqs hooks directly
    const [, setEntityId] = useQueryState(
      "entityId",
      entityParams.entityId.withOptions({
        startTransition,
        shallow: true, // This is key - shallow:true preserves other URL params
      }),
    );

    const [, setModalType] = useQueryState(
      "modal",
      entityParams.modal.withOptions({
        startTransition,
        shallow: true, // This is key - shallow:true preserves other URL params
      }),
    );

    // Create a click handler for opening the modal
    const handleClick = useCallback(
      (e: React.MouseEvent) => {
        e.preventDefault();
        // Set both parameters with shallow:true to preserve page and pageSize
        Promise.all([
          setEntityId(id || "", { shallow: true }),
          setModalType("edit", { shallow: true }),
        ]).catch(console.error);
      },
      [id, setEntityId, setModalType],
    );

    return (
      <span
        onClick={handleClick}
        className={`${className || ""} cursor-pointer`}
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
      </span>
    );
  },
);

// Memoized SecondaryInfoLink component
const SecondaryInfoLink = memo(
  ({
    id,
    displayText,
    clickable,
  }: {
    id: string | undefined;
    displayText: string;
    clickable: boolean;
  }) => {
    const [, startTransition] = useTransition();

    // Use the nuqs hooks directly
    const [, setEntityId] = useQueryState(
      "entityId",
      entityParams.entityId.withOptions({
        startTransition,
        shallow: true,
      }),
    );

    const [, setModalType] = useQueryState(
      "modal",
      entityParams.modal.withOptions({
        startTransition,
        shallow: true,
      }),
    );

    // Create a click handler for opening the modal
    const handleClick = useCallback(
      (e: React.MouseEvent) => {
        e.preventDefault();
        // Set both parameters with shallow:true to preserve page and pageSize
        Promise.all([
          setEntityId(id || "", { shallow: true }),
          setModalType("edit", { shallow: true }),
        ]).catch(console.error);
      },
      [id, setEntityId, setModalType],
    );

    if (!clickable) {
      return <p>{displayText}</p>;
    }

    return (
      <span
        onClick={handleClick}
        className="text-2xs text-muted-foreground underline hover:text-muted-foreground/70 cursor-pointer"
      >
        {displayText}
      </span>
    );
  },
);

// Define the EntityRefCell component type first, then memoize it
interface EntityRefCellProps<TEntity, TParent> {
  entity: TEntity;
  config: EntityRefConfig<TEntity, TParent>;
  parent: TParent;
}

function EntityRefCellBase<TEntity, TParent extends Record<string, any>>(
  props: EntityRefCellProps<TEntity, TParent>,
) {
  const { entity, config, parent } = props;

  if (!entity) {
    return <p className="text-muted-foreground">-</p>;
  }

  const id = config.getId(entity);
  const displayText = config.getDisplayText(entity);
  const secondaryInfo = config.getSecondaryInfo?.(entity, parent);
  const color = config.color?.getColor(entity);

  // clickable should default to true unless otherwise specified
  const clickable = secondaryInfo?.clickable ?? true;

  return (
    <div className="flex flex-col gap-0.5">
      <Tooltip delayDuration={300}>
        <TooltipTrigger asChild>
          <EntityRefLink
            id={id}
            displayText={displayText}
            className={config.className}
            color={color}
          />
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
              <SecondaryInfoLink
                id={config.getId(secondaryInfo.entity)}
                displayText={secondaryInfo.displayText}
                clickable={clickable}
              />
            </TooltipTrigger>
            <TooltipContent>
              <p>Click to view {secondaryInfo.displayText}</p>
            </TooltipContent>
          </Tooltip>
        </div>
      )}
    </div>
  );
}

const EntityRefCell = memo(EntityRefCellBase) as typeof EntityRefCellBase;

// Define the NestedEntityRefCell component type first, then memoize it
interface NestedEntityRefCellProps<TEntity, TParent> {
  getValue: () => TEntity | null | undefined;
  row: { original: TParent };
  config: NestedEntityRefConfig<TEntity, TParent>;
}

function NestedEntityRefCellBase<TEntity, TParent extends Record<string, any>>(
  props: NestedEntityRefCellProps<TEntity, TParent>,
) {
  const { getValue, row, config } = props;
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
          <EntityRefLink
            id={id}
            displayText={displayText}
            className={config.className}
            color={color}
          />
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
                <SecondaryInfoLink
                  id={config.getId(secondaryInfo.entity)}
                  displayText={secondaryInfo.displayText}
                  clickable={clickable}
                />
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
}

const NestedEntityRefCell = memo(
  NestedEntityRefCellBase,
) as typeof NestedEntityRefCellBase;

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
    header: config.getHeaderText ?? "",
    cell: ({ getValue, row }) => {
      const entity = getValue();
      if (!entity) {
        return <p className="text-muted-foreground">-</p>;
      }
      return (
        <EntityRefCell<NonNullable<TValue>, T>
          entity={entity as NonNullable<TValue>}
          config={config}
          parent={row.original}
        />
      );
    },
  }) as ColumnDef<T>;
}

export function createEntityColumn<T extends Record<string, any>>(
  columnHelper: ColumnHelper<T>,
  accessorKey: keyof T,
  config: EntityColumnConfig<T, keyof T>,
): ColumnDef<T> {
  return columnHelper.accessor((row) => row[accessorKey], {
    id: accessorKey as string,
    header: config.getHeaderText ?? "",
    cell: ({ row }) => {
      const entity = row.original;

      if (!entity) {
        return <p>-</p>;
      }

      const id = config.getId(row.original);
      const displayText = config.getDisplayText(row.original);
      const color = config.getColor?.(row.original);

      return (
        <EntityRefLink
          id={id}
          displayText={displayText}
          className={config.className}
          color={color}
        />
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
    header: config.getHeaderText ?? "",
    cell: (info) => (
      <NestedEntityRefCell<TValue, T> {...info} config={config} />
    ),
  }) as ColumnDef<T>;
}
