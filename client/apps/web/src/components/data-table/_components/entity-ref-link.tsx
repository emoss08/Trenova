import { cn } from "@/lib/utils";
import { memo } from "react";
import { Link } from "react-router";

export type EntityRefConfig<TEntity, TParent> = {
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
    basePath?: string;
  } | null;
  className?: string;
  color?: {
    getColor: (entity: TEntity) => string | undefined;
  };
};

type EntityRefLinkInnerProps = React.ComponentProps<typeof Link>;

export function EntityRefLinkInner({ className, children, ...props }: EntityRefLinkInnerProps) {
  return (
    <Link {...props} className={cn("max-w-px cursor-pointer text-left", className)}>
      {children}
    </Link>
  );
}

export function EntityRefLinkDisplayText({ children }: { children: React.ReactNode }) {
  return (
    <span className="text-sm font-normal text-nowrap underline hover:text-foreground/70">
      {children}
    </span>
  );
}

export function EntityRefLinkColor({ color, displayText }: { color: string; displayText: string }) {
  return (
    <EntityRefLinkColorInner>
      <div
        className="size-2 shrink-0 rounded-full"
        style={{
          backgroundColor: color,
        }}
      />
      <p>{displayText}</p>
    </EntityRefLinkColorInner>
  );
}

const EntityRefLinkColorInner = memo(function EntityRefLinkColorInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center gap-x-1.5 text-sm font-normal text-foreground underline hover:text-foreground/70">
      {children}
    </div>
  );
});

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
    const to = `${basePath ?? ""}?panelEntityId=${id}&panelType=edit`;

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

export type EntityColumnConfig<T extends Record<string, any>, K extends keyof T> = {
  accessorKey: K;
  getHeaderText?: string;
  getId: (entity: T) => string | undefined;
  getDisplayText: (entity: T) => string;
  className?: string;
  getColor?: (entity: T) => string | undefined;
};

// Define the EntityRefCell component type first, then memoize it
interface EntityRefCellProps<TEntity, TParent> {
  entity: TEntity;
  config: EntityRefConfig<TEntity, TParent>;
  parent: TParent;
}

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
        className="cursor-pointer text-2xs text-foreground underline hover:text-foreground/70"
        title={`Click to view ${displayText}`}
      >
        {displayText}
      </Link>
    );
  },
);

export function EntityRefCell<TEntity, TParent extends Record<string, any>>(
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
        <div className="flex items-center gap-1 text-2xs text-muted-foreground">
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

export type NestedEntityRefConfig<TEntity, TParent> = EntityRefConfig<TEntity, TParent> & {
  getEntity: (parent: TParent) => TEntity | null | undefined;
  columnId?: string;
};

interface NestedEntityRefCellProps<TEntity, TParent> {
  getValue: () => TEntity | null | undefined;
  row: { original: TParent };
  config: NestedEntityRefConfig<TEntity, TParent>;
}
export function NestedEntityRefCell<TEntity, TParent extends Record<string, any>>(
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
        <div className="flex items-center gap-1 text-2xs text-muted-foreground">
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
