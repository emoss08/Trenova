import type { VariableValueType } from "@trenova/shared/types/formula-template";

/** A typed default as the template stores it: a number for Number, a flag for Boolean, text for String. */
export function coerceVariableDefaultValue(type: VariableValueType, raw: unknown): unknown {
  if (raw === "" || raw === null || raw === undefined) return undefined;

  switch (type) {
    case "Number": {
      if (typeof raw === "number") return raw;
      if (typeof raw !== "string") return raw;
      const parsed = Number(raw);
      return Number.isNaN(parsed) ? raw : parsed;
    }
    case "Boolean": {
      if (typeof raw === "boolean") return raw;
      if (raw === "true") return true;
      if (raw === "false") return false;
      return raw;
    }
    case "String":
      if (typeof raw === "string") return raw;
      if (typeof raw === "number" || typeof raw === "boolean") return String(raw);
      return raw;
    default:
      return raw;
  }
}
