import type { OperationDefinition } from "@/lib/role-api";

export function impliedOperations(
  granted: readonly string[],
  definitions: readonly OperationDefinition[],
): Map<string, string[]> {
  const held = new Set(granted);
  const implied = new Map<string, string[]>();

  for (const definition of definitions) {
    if (!held.has(definition.operation)) {
      continue;
    }
    for (const dependency of definition.implies ?? []) {
      if (held.has(dependency)) {
        continue;
      }
      const grantedBy = implied.get(dependency) ?? [];
      grantedBy.push(definition.operation);
      implied.set(dependency, grantedBy);
    }
  }

  return implied;
}
