/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { cn } from "@/lib/utils";
import { parseAsString, useQueryState } from "nuqs";
import { memo, useCallback, useTransition } from "react";
import {
  EntityRefConfig,
  NestedEntityRefConfig,
} from "./data-table-column-types";

// Entity parameter definitions - same as in data-table.tsx
const entityParams = {
  entityId: parseAsString,
  modal: parseAsString,
};

// Memoized EntityRefLink component to avoid re-renders
export const EntityRefLink = memo(
  ({
    id,
    displayText,
    className,
    color,
  }: {
    basePath: string;
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
        // Set both parameters with shallow:true
        Promise.all([
          setEntityId(id || "", { shallow: true }),
          setModalType("edit", { shallow: true }),
        ]).catch(console.error);
      },
      [id, setEntityId, setModalType],
    );

    return (
      <span onClick={handleClick} className={cn("cursor-pointer", className)}>
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
EntityRefLink.displayName = "EntityRefLink";

// Memoized SecondaryInfoLink component
export const SecondaryInfoLink = memo(
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
        // Set both parameters with shallow:true
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
SecondaryInfoLink.displayName = "SecondaryInfoLink";

// Memoized EntityRefCell component
export const EntityRefCell = memo(
  <T extends Record<string, any>, K extends keyof T, TValue = T[K]>({
    entity,
    config,
    parent,
  }: {
    entity: NonNullable<TValue>;
    config: EntityRefConfig<NonNullable<TValue>, T>;
    parent: T;
  }) => {
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
        <EntityRefLink
          basePath={config.basePath}
          id={id}
          displayText={displayText}
          className={config.className}
          color={color}
        />

        {secondaryInfo && (
          <div className="flex items-center gap-1 text-muted-foreground text-2xs">
            {secondaryInfo.label && <span>{secondaryInfo.label}:</span>}
            <SecondaryInfoLink
              id={config.getId(secondaryInfo.entity)}
              displayText={secondaryInfo.displayText}
              clickable={clickable}
            />
          </div>
        )}
      </div>
    );
  },
);
EntityRefCell.displayName = "EntityRefCell";

// Memoized NestedEntityRefCell component
export const NestedEntityRefCell = memo(
  <T extends Record<string, any>, TValue>({
    getValue,
    row,
    config,
  }: {
    getValue: () => TValue;
    row: { original: T };
    config: NestedEntityRefConfig<TValue, T>;
  }) => {
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
        <EntityRefLink
          basePath={config.basePath}
          id={id}
          displayText={displayText}
          className={config.className}
          color={color}
        />

        {secondaryInfo && (
          <div className="flex items-center gap-1 text-muted-foreground text-2xs">
            {secondaryInfo.label && <span>{secondaryInfo.label}:</span>}
            {clickable ? (
              <SecondaryInfoLink
                id={config.getId(secondaryInfo.entity)}
                displayText={secondaryInfo.displayText}
                clickable={clickable}
              />
            ) : (
              <p>{secondaryInfo.displayText}</p>
            )}
          </div>
        )}
      </div>
    );
  },
);
NestedEntityRefCell.displayName = "NestedEntityRefCell";
