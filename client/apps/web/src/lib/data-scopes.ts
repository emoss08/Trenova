import type { DataScope } from "@trenova/shared/types/role";

export type ScopeChoice = {
  value: DataScope;
  enforced: boolean;
};

const scopeOrder: readonly DataScope[] = ["own", "team", "organization", "business_unit", "all"];

export function scopeChoices(
  enforceable: readonly DataScope[] | null | undefined,
  current: DataScope | undefined,
): ScopeChoice[] {
  const offered = new Set<DataScope>(enforceable?.length ? enforceable : ["organization"]);
  const values = new Set<DataScope>(offered);
  if (current) {
    values.add(current);
  }

  return scopeOrder
    .filter((scope) => values.has(scope))
    .map((scope) => ({ value: scope, enforced: offered.has(scope) }));
}

export function isScopeEnforced(
  enforceable: readonly DataScope[] | null | undefined,
  current: DataScope,
): boolean {
  return scopeChoices(enforceable, current).some(
    (choice) => choice.value === current && choice.enforced,
  );
}
