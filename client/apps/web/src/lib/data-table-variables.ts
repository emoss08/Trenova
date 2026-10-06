import type {
  DataTableGraphQLSource,
  DataTableQueryOptions,
} from "@trenova/shared/types/data-table";

export function resolveExtraVariables<TData extends Record<string, unknown>>(
  config: DataTableGraphQLSource<TData>,
  pageSize: number,
  options?: DataTableQueryOptions,
): Record<string, unknown> {
  if (!config.extraVariables) {
    return {};
  }

  if (typeof config.extraVariables === "function") {
    return config.extraVariables({ pageSize, options });
  }

  return config.extraVariables;
}

export function resolveInputExtraVariables<TData extends Record<string, unknown>>(
  config: DataTableGraphQLSource<TData>,
  pageSize: number,
  options?: DataTableQueryOptions,
): Record<string, unknown> {
  if (!config.inputExtraVariables) {
    return {};
  }

  if (typeof config.inputExtraVariables === "function") {
    return config.inputExtraVariables({ pageSize, options });
  }

  return config.inputExtraVariables;
}

/** The variables a config adds beyond the table's own, resolved for one request. */
export function resolveGraphQLVariableSources<TData extends Record<string, unknown>>(
  config: DataTableGraphQLSource<TData>,
  pageSize: number,
  options?: DataTableQueryOptions,
) {
  return {
    extraVariables: resolveExtraVariables(config, pageSize, options),
    inputExtraVariables: resolveInputExtraVariables(config, pageSize, options),
  };
}
