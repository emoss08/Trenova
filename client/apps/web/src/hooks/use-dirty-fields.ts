import { get, useFormState, type Control, type FieldPath, type FieldValues } from "react-hook-form";

/*
 * react-hook-form changes its dirtyFields object in place, so the object keeps one
 * identity for the life of the form. The React Compiler memoizes anything derived from
 * it on that identity, and the derivation never runs again: a save dock that hides
 * while nothing has changed never opens. Each hook here opts out of memoization, so
 * what it derives is read afresh on every render and handed on as a new value.
 */

/** The top-level fields that differ from what the form opened with, less its version. */
export function useChangedFieldNames<T extends FieldValues>(
  control: Control<T, any, any>,
  versionField: string = "version",
): string[] {
  "use no memo";
  const { dirtyFields } = useFormState({ control });
  return Object.keys(dirtyFields).filter((key) => key !== versionField);
}

/** Whether any of the named fields, at any depth, differs from what the form opened with. */
export function useAnyFieldDirty<T extends FieldValues>(
  control: Control<T, any, any>,
  names: readonly FieldPath<T>[],
): boolean {
  "use no memo";
  const { dirtyFields } = useFormState({
    control,
    name: names,
    disabled: names.length === 0,
  });
  return names.some((name) => Boolean(get(dirtyFields, name)));
}
