import {
  get,
  type DeepPartial,
  type FieldValues,
  type Path,
  type UseFormReturn,
} from "react-hook-form";

/**
 * Whether a fetched record is newer than the one a form is seated on.
 *
 * A save reseats the form on the record the server returned, but the query cache still holds
 * the record from before the save until its refetch lands. Treating any version that differs
 * as a replacement would put that older record back in the form, and the editor would show the
 * source as it was before the save. Only a newer version, or the first one, counts.
 */
export function isNewerRecord(
  recordVersion: number | null | undefined,
  seatedVersion: number | null,
): boolean {
  return seatedVersion === null || (recordVersion ?? 0) > seatedVersion;
}

type ReseatKeepingEditsParams<T extends FieldValues> = {
  form: UseFormReturn<T>;
  /** The form's values for the record to seat on. */
  values: T;
  /** The values the person's edits are measured from. */
  base: DeepPartial<T>;
  /** The fields the person edits. */
  fields: readonly Path<T>[];
};

/**
 * Reseats a form on a record without losing the person's edits: a field whose value differs
 * from `base` keeps it and stays dirty against the new record, and every other field takes the
 * record's value.
 */
export function reseatKeepingEdits<T extends FieldValues>({
  form,
  values,
  base,
  fields,
}: ReseatKeepingEditsParams<T>): void {
  const typed = fields.map((field) => [field, form.getValues(field)] as const);
  form.reset(values);
  for (const [field, value] of typed) {
    if (value !== get(base, field)) {
      form.setValue(field, value, { shouldDirty: true });
    }
  }
}
