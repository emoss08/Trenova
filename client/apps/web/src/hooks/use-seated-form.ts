import { isNewerRecord, reseatKeepingEdits } from "@/lib/seated-record";
import { useEffect, useState } from "react";
import { useFormState, type FieldValues, type Path, type UseFormReturn } from "react-hook-form";

/** A record as the form holds it: its values and the version a save is checked against. */
export type SeatedRecord<T extends FieldValues> = {
  values: T;
  version: number;
};

type UseSeatedFormParams<T extends FieldValues> = {
  form: UseFormReturn<T>;
  /** The record as the server last sent it; null while it loads. */
  record: SeatedRecord<T> | null;
  /** The fields the person edits and a save writes. */
  fields: readonly Path<T>[];
};

export type SeatedForm<T extends FieldValues> = {
  isSeated: boolean;
  /** The version the form is seated on, which a save sends; null until the record loads. */
  version: number | null;
  /**
   * A newer record the server holds, kept out of the form because the form has edits. A save
   * would be refused while it is held, so the person decides first.
   */
  held: SeatedRecord<T> | null;
  /** Seats the form on a record, dropping any edits: a rollback, or loading the held record. */
  seat: (record: SeatedRecord<T>) => void;
  /** Seats the form on what a save returned, keeping what was typed while it was in flight. */
  seatSaved: (saved: SeatedRecord<T>, sent: T) => void;
  /** Seats the form on the held record with the person's edits kept over it. */
  keepEdits: () => void;
};

/**
 * Seats an edit page's form on a versioned record and keeps it there. A newer record replaces
 * the form only while the form has no edits; with edits, it is held for the person to compare.
 * An older one (the cache before a save's refetch lands) never does.
 */
export function useSeatedForm<T extends FieldValues>({
  form,
  record,
  fields,
}: UseSeatedFormParams<T>): SeatedForm<T> {
  const [version, setVersion] = useState<number | null>(null);
  const { isDirty } = useFormState({ control: form.control });
  const newer = record !== null && isNewerRecord(record.version, version);
  const replaces = newer && (version === null || !isDirty);

  useEffect(() => {
    if (record && replaces) {
      form.reset(record.values);
      // oxlint-disable-next-line react/set-state-in-effect -- form.reset must run before the version lets the studio mount, so both stay in the effect
      setVersion(record.version);
    }
  }, [record, replaces, form]);

  const seat = (next: SeatedRecord<T>) => {
    form.reset(next.values);
    setVersion(next.version);
  };

  return {
    isSeated: version !== null,
    version,
    held: newer && !replaces ? record : null,
    seat,
    seatSaved: (saved, sent) => {
      reseatKeepingEdits({ form, values: saved.values, base: sent, fields });
      setVersion(saved.version);
    },
    keepEdits: () => {
      if (!newer || replaces || !record) return;
      reseatKeepingEdits({
        form,
        values: record.values,
        base: form.formState.defaultValues ?? {},
        fields,
      });
      setVersion(record.version);
    },
  };
}
