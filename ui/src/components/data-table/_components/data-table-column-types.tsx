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
