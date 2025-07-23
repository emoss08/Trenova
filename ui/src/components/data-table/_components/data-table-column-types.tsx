/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

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

export type NestedEntityRefConfig<TEntity, TParent> = EntityRefConfig<
  TEntity,
  TParent
> & {
  getEntity: (parent: TParent) => TEntity | null | undefined;
  columnId?: string;
};

export type EntityColumnConfig<
  T extends Record<string, any>,
  K extends keyof T,
> = {
  accessorKey: K;
  getHeaderText?: string;
  getId: (entity: T) => string | undefined;
  getDisplayText: (entity: T) => string;
  className?: string;
  getColor?: (entity: T) => string | undefined;
};
