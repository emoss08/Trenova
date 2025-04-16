import { InternalLink } from "@/components/ui/link";
import {
    Tooltip,
    TooltipContent,
    TooltipTrigger,
} from "@/components/ui/tooltip";
import { memo, useMemo } from "react";
import { EntityRefConfig, NestedEntityRefConfig } from "./data-table-column-types";

// Memoized EntityRefLink component to avoid re-renders
export const EntityRefLink = memo(
  ({
    basePath,
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
    // Create search params object once
    const linkTo = useMemo(
      () => ({
        pathname: basePath,
        search: `?entityId=${id}&modal=edit`,
      }),
      [basePath, id]
    );

    const linkState = useMemo(
      () => ({
        isNavigatingToModal: true,
      }),
      []
    );

    return (
      <InternalLink
        to={linkTo}
        state={linkState}
        className={className}
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
    );
  }
);
EntityRefLink.displayName = "EntityRefLink";

// Memoized SecondaryInfoLink component
export const SecondaryInfoLink = memo(
  ({
    basePath,
    id,
    displayText,
    clickable,
  }: {
    basePath: string;
    id: string | undefined;
    displayText: string;
    clickable: boolean;
  }) => {
    // Create search params object once
    const linkTo = useMemo(
      () => ({
        pathname: basePath,
        search: `?entityId=${id}&modal=edit`,
      }),
      [basePath, id]
    );

    const linkState = useMemo(
      () => ({
        isNavigatingToModal: true,
      }),
      []
    );

    if (!clickable) {
      return <p>{displayText}</p>;
    }

    return (
      <InternalLink
        to={linkTo}
        state={linkState}
        className="text-2xs text-muted-foreground underline hover:text-muted-foreground/70"
        replace
        preventScrollReset
        viewTransition
      >
        {displayText}
      </InternalLink>
    );
  }
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
        <Tooltip delayDuration={300}>
          <TooltipTrigger asChild>
            <EntityRefLink
              basePath={config.basePath}
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
                  basePath={config.basePath}
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
        <Tooltip delayDuration={300}>
          <TooltipTrigger asChild>
            <EntityRefLink
              basePath={config.basePath}
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
                    basePath={config.basePath}
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
);
NestedEntityRefCell.displayName = "NestedEntityRefCell"; 