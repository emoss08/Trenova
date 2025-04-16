export const integrationKeys = {
  all: () => [{ scope: "integrations" }] as const,
  lists: () => [{ ...integrationKeys.all()[0], entity: "list" }] as const,
  list: (params: Record<string, any> = {}) =>
    [{ ...integrationKeys.lists()[0], ...params }] as const,
  details: () => [{ ...integrationKeys.all()[0], entity: "detail" }] as const,
  detail: (id: string) => [{ ...integrationKeys.details()[0], id }] as const,
  configs: () => [{ ...integrationKeys.all()[0], entity: "config" }] as const,
  config: (id: string) => [{ ...integrationKeys.configs()[0], id }] as const,
};
