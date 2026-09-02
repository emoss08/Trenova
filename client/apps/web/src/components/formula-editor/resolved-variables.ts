type Scalar = string | number | boolean;

function isScalar(value: unknown): value is Scalar {
  return typeof value === "string" || typeof value === "number" || typeof value === "boolean";
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

/**
 * Flattens the nested variables a real-shipment preview resolves into the
 * dotted keys the sample-data editor and scenarios use (`customer.name`),
 * keeping only scalar values. Nulls, arrays, and functions are dropped: they
 * are not values a person would type into a scenario.
 */
export function flattenResolvedVariables(
  variables: Record<string, unknown>,
  prefix = "",
): Record<string, Scalar> {
  const flat: Record<string, Scalar> = {};

  for (const [key, value] of Object.entries(variables)) {
    const path = prefix ? `${prefix}.${key}` : key;
    if (isScalar(value)) {
      flat[path] = value;
    } else if (isPlainObject(value)) {
      Object.assign(flat, flattenResolvedVariables(value, path));
    }
  }

  return flat;
}
