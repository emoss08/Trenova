import { FieldValues, Path } from "react-hook-form";

/**
 * Gets a nested value from an object using a path string
 * @param obj The object to traverse
 * @param path The path string (e.g., "profile.dob")
 * @returns The value at the path or undefined if not found
 */
function getNestedValue(obj: any, path: string): any {
  return path.split(".").reduce((current, part) => {
    return current && typeof current === "object" ? current[part] : undefined;
  }, obj);
}

/**
 * Checks if there are any errors in the specified form fields
 * @param errors The form errors object from react-hook-form
 * @param fields Array of field paths to check for errors
 * @returns boolean indicating if any of the specified fields have errors
 */
export function checkSectionErrors<T extends FieldValues>(
  errors: Partial<T>,
  fields: Path<T>[],
): boolean {
  return fields.some((field) => {
    // Get the nested error value
    const error = getNestedValue(errors, field as string);

    // Check if there's an error message or object
    return !!error;
  });
}

/**
 * Type-safe version that provides more detailed error information
 */
export function getSectionErrors<T extends FieldValues>(
  errors: Partial<T>,
  fields: Path<T>[],
): { hasErrors: boolean; errorFields: Path<T>[] } {
  const errorFields = fields.filter((field) => {
    const error = getNestedValue(errors, field as string);
    return !!error;
  });

  return {
    hasErrors: errorFields.length > 0,
    errorFields,
  };
}

// Example usage in debugging:
export function debugFormErrors<T extends FieldValues>(
  errors: Partial<T>,
  fields: Path<T>[],
): void {
  console.group("Form Section Errors Debug");
  fields.forEach((field) => {
    const error = getNestedValue(errors, field as string);
    console.log(`Field: ${field}`, { hasError: !!error, errorValue: error });
  });
  console.groupEnd();
}
