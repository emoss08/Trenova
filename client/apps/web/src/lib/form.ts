import type { FieldErrors, FieldValues, Path } from "react-hook-form";

/**
 * Gets a nested value from an object using a path string
 * @param obj The object to traverse
 * @param path The path string (e.g., "profile.dob")
 * @returns The value at the path or undefined if not found
 */
function getNestedValue(obj: unknown, path: string): unknown {
  return path.split(".").reduce((current, part) => {
    return current && typeof current === "object"
      ? (current as Record<string, unknown>)[part]
      : undefined;
  }, obj);
}

/**
 * Checks if there are any errors in the specified form fields
 * @param errors The form errors object from react-hook-form
 * @param fields Array of field paths to check for errors
 * @returns boolean indicating if any of the specified fields have errors
 */
export function checkSectionErrors<T extends FieldValues>(
  errors: FieldErrors<T>,
  fields: Path<T>[],
): boolean {
  return fields.some((field) => {
    const error = getNestedValue(errors, field as string);
    return !!error;
  });
}
