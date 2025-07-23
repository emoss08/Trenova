/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

/* eslint-disable react-refresh/only-export-components */
/* eslint-disable react/display-name */

import {
  EntityRefLinkColor,
  EntityRefLinkDisplayText,
  EntityRefLinkInner,
} from "@/components/entity-refs/entity-ref-link";
import { StatusBadge } from "@/components/status-badge";
import type { Status } from "@/types/common";
import { ColumnDef, ColumnHelper, type Row } from "@tanstack/react-table";
import { memo } from "react";
import { Link } from "react-router";
import { v4 } from "uuid";
import {
  EntityColumnConfig,
  EntityRefConfig,
  NestedEntityRefConfig,
} from "./data-table-column-types";
import {
  DataTableDescription,
  HoverCardTimestamp,
} from "./data-table-components";

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
    const to = `${basePath ?? ""}?entityId=${id}&modalType=edit`;

    return (
      <EntityRefLinkInner
        to={to}
        target="_blank"
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
    if (!clickable) {
      return <p>{displayText}</p>;
    }

    const to = `${basePath ?? ""}?entityId=${id}&modalType=edit`;

    return (
      <Link
        to={to}
        target="_blank"
        className="text-2xs text-foreground underline hover:text-foreground/70 cursor-pointer"
        title={`Click to view ${displayText}`}
      >
        {displayText}
      </Link>
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
    <div className="flex flex-col gap-0.5 truncate">
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

export const EntityRefCell = memo(
  EntityRefCellBase,
) as typeof EntityRefCellBase;

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
    <div className="flex flex-col">
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

export const NestedEntityRefCell = memo(
  NestedEntityRefCellBase,
) as typeof NestedEntityRefCellBase;

export function createCommonColumns<T extends Record<string, unknown>>() {
  return {
    status: {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }: { row: Row<T> }) => {
        const status = row.getValue<Status | undefined>("status");
        return <StatusBadge status={status as Status} />;
      },
    },
    description: {
      accessorKey: "description",
      header: "Description",
      cell: ({ row }: { row: Row<T> }) => {
        const description = row.getValue<string | undefined>("description");
        return (
          <DataTableDescription
            description={description as string | undefined}
          />
        );
      },
    },
    createdAt: {
      accessorKey: "createdAt",
      header: "Created At",
      cell: ({ row }: { row: Row<T> }) => {
        const createdAt = row.getValue<number | undefined>("createdAt");
        return <HoverCardTimestamp timestamp={createdAt} />;
      },
    },
  };
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
