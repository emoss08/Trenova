import { isRecordEntityType, recordPath } from "@/config/record-links";
import { LockedReference } from "@trenova/shared/components/withheld-value";
import { useCanReadKind } from "@trenova/shared/hooks/use-can-read-kind";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReferenceKind } from "@trenova/shared/types/generated/reference-views";
import { cn } from "@trenova/shared/lib/utils";
import { memo } from "react";
import { Link } from "react-router";

export type EntityRefConfig<TEntity, TParent> = {
  /**
   * The referenced record's kind. Given, the cell links to the record only for someone with
   * the kind's read and otherwise draws the name with a lock
   * (docs/engineering/reference-views.md). A kind with a page in `RECORD_LINKS` opens there;
   * any other opens at `basePath`.
   */
  kind?: ReferenceKind;
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
    <span className="hover:text-foreground/70 text-sm font-normal text-nowrap underline">
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
    <div className="text-foreground hover:text-foreground/70 flex items-center gap-x-1.5 text-sm font-normal underline">
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
    kind,
  }: {
    id: string | undefined;
    displayText: string;
    className?: string;
    color?: string;
    basePath?: string;
    kind?: ReferenceKind;
  }) => {
    const t = useT();
    const canRead = useCanReadKind(kind ?? null);

    if (kind && !canRead) {
      return (
        <LockedReference kind={kind} className={cn("text-sm", className)}>
          {displayText}
        </LockedReference>
      );
    }
    if (kind && !id) {
      return <span className={cn("text-sm", className)}>{displayText}</span>;
    }

    const to =
      kind && id && isRecordEntityType(kind)
        ? recordPath(kind, id)
        : `${basePath ?? ""}?panelEntityId=${id}&panelType=edit`;

    return (
      <EntityRefLinkInner
        to={to}
        target="_blank"
        className={`${className || ""} cursor-pointer`}
        title={t("Click to view {0}", displayText)}
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
    const t = useT();
    if (!clickable) {
      return <p>{displayText}</p>;
    }

    const to = `${basePath ?? ""}?entityId=${id}&modalType=edit`;

    return (
      <Link
        to={to}
        target="_blank"
        className="text-2xs text-foreground hover:text-foreground/70 cursor-pointer underline"
        title={t("Click to view {0}", displayText)}
      >
        {displayText}
      </Link>
    );
  },
);

export function EntityRefCell<TEntity, TParent extends Record<string, any>>(
  props: EntityRefCellProps<TEntity, TParent>,
) {
  const t = useT();

  const { entity, config, parent } = props;

  if (!entity) {
    return <p className="text-muted-foreground">—</p>;
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
        kind={config.kind}
      />
      {secondaryInfo && (
        <div className="text-2xs text-muted-foreground flex items-center gap-1">
          {secondaryInfo.label && <span>{t(secondaryInfo.label)}:</span>}
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
  const t = useT();

  const { getValue, row, config } = props;
  const entity = getValue();

  if (!entity) {
    return <p className="text-muted-foreground">—</p>;
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
        kind={config.kind}
      />

      {secondaryInfo && (
        <div className="text-2xs text-muted-foreground flex items-center gap-1">
          {secondaryInfo.label && <span>{t(secondaryInfo.label)}:</span>}
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
