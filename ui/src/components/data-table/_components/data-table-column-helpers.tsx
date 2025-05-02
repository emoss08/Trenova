/* eslint-disable react-refresh/only-export-components */
/* eslint-disable react/display-name */

import {
  EntityRefLinkColor,
  EntityRefLinkDisplayText,
  EntityRefLinkInner,
} from "@/components/entity-refs/entity-ref-link";
import { StatusBadge } from "@/components/status-badge";
import { generateDateOnlyString, toDate } from "@/lib/date";
import type { Status } from "@/types/common";
import { ColumnDef, ColumnHelper } from "@tanstack/react-table";
import { parseAsString, useQueryState } from "nuqs";
import { memo, useCallback, useTransition } from "react";
import { useNavigate } from "react-router";
import { v4 } from "uuid";
import {
  EntityColumnConfig,
  EntityRefConfig,
  NestedEntityRefConfig,
} from "./data-table-column-types";
import { DataTableDescription } from "./data-table-components";

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
    basePath,
  }: {
    id: string | undefined;
    displayText: string;
    className?: string;
    color?: string;
    basePath?: string;
  }) => {
    const [, startTransition] = useTransition();
    const navigate = useNavigate();

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
        if (basePath && id) {
          // If basePath is provided, navigate to the base URL first, then set params after navigation
          navigate(basePath, {
            replace: true, // Use replace to avoid adding to history stack
            state: { pendingEntityId: id }, // Pass the entity ID through state
          });
        } else {
          // Otherwise use the existing behavior with URL params
          Promise.all([
            setEntityId(id || "", { shallow: true }),
            setModalType("edit", { shallow: true }),
          ]).catch(console.error);
        }
      },
      [id, setEntityId, setModalType, basePath, navigate],
    );

    return (
      <EntityRefLinkInner
        onClick={handleClick}
        className={`${className || ""} cursor-pointer`}
        title={`Click to view ${displayText}`}
      >
        {color ? (
          <EntityRefLinkColor color={color} displayText={displayText} />
        ) : (
          <EntityRefLinkDisplayText>{displayText}</EntityRefLinkDisplayText>
        )}
      </EntityRefLinkInner>
    );
  },
);

// Memoized SecondaryInfoLink component
const SecondaryInfoLink = memo(
  ({
    id,
    displayText,
    clickable,
    basePath,
  }: {
    id: string | undefined;
    displayText: string;
    clickable: boolean;
    basePath?: string;
  }) => {
    const [, startTransition] = useTransition();
    const navigate = useNavigate();

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
        if (basePath && id) {
          // If basePath is provided, navigate to the base URL first, then set params after navigation
          navigate(basePath, {
            replace: true, // Use replace to avoid adding to history stack
            state: { pendingEntityId: id }, // Pass the entity ID through state
          });
        } else {
          // Set both parameters with shallow:true to preserve page and pageSize
          Promise.all([
            setEntityId(id || "", { shallow: true }),
            setModalType("edit", { shallow: true }),
          ]).catch(console.error);
        }
      },
      [id, setEntityId, setModalType, basePath, navigate],
    );

    if (!clickable) {
      return <p>{displayText}</p>;
    }

    return (
      <span
        onClick={handleClick}
        className="text-2xs text-foreground underline hover:text-foreground/70 cursor-pointer"
        title={`Click to view ${displayText}`}
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
  const basePath = config.basePath;

  // clickable should default to true unless otherwise specified
  const clickable = secondaryInfo?.clickable ?? true;

  return (
    <div className="flex flex-col gap-0.5">
      <EntityRefLink
        id={id}
        displayText={displayText}
        className={config.className}
        color={color}
        basePath={basePath}
      />
      {secondaryInfo && (
        <div className="flex items-center gap-1 text-muted-foreground text-2xs">
          {secondaryInfo.label && <span>{secondaryInfo.label}:</span>}
          <SecondaryInfoLink
            id={config.getId(secondaryInfo.entity)}
            displayText={secondaryInfo.displayText}
            clickable={clickable}
            basePath={secondaryInfo.basePath || basePath}
          />
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
  const basePath = config.basePath;

  // clickable should default to true unless otherwise specified
  const clickable = secondaryInfo?.clickable ?? true;

  return (
    <div className="flex flex-col gap-0.5">
      <EntityRefLink
        id={id}
        displayText={displayText}
        className={config.className}
        color={color}
        basePath={basePath}
      />

      {secondaryInfo && (
        <div className="flex items-center gap-1 text-muted-foreground text-2xs">
          {secondaryInfo.label && <span>{secondaryInfo.label}:</span>}
          {clickable ? (
            <SecondaryInfoLink
              id={config.getId(secondaryInfo.entity)}
              displayText={secondaryInfo.displayText}
              clickable={clickable}
              basePath={secondaryInfo.basePath || basePath}
            />
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
    status: columnHelper.display({
      id: "status",
      header: "Status",
      cell: ({ row }) => {
        const status = row.original.status;
        return <StatusBadge status={status as Status} />;
      },
    }),
    description: columnHelper.display({
      id: "description",
      header: "Description",
      cell: ({ row }) => (
        <DataTableDescription
          description={row.original.description as string | undefined}
        />
      ),
    }),

    createdAt: createdAtColumn(columnHelper) as ColumnDef<T>,
  };
}

function createdAtColumn<T extends Record<string, unknown>>(
  columnHelper: ColumnHelper<T>,
) {
  return columnHelper.display({
    id: "createdAt",
    header: "Created At",
    cell: ({ row }) => {
      const { createdAt } = row.original;
      const date = toDate(createdAt as number);
      if (!date) return <p>-</p>;

      return <p>{generateDateOnlyString(date)}</p>;
    },
  }) as ColumnDef<T>;
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
  return columnHelper.display({
    id: accessorKey as string,
    header: config.getHeaderText ?? "",
    cell: ({ row }) => {
      const entity = row.original[accessorKey];

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
  return columnHelper.display({
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
