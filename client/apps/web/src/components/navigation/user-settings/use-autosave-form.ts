import { handleMutationError } from "@/hooks/use-api-mutation";
import { translate } from "@trenova/shared/i18n/runtime";
import { useCallback, useEffect, useLayoutEffect, useRef } from "react";
import type { FieldValues, UseFormReturn } from "react-hook-form";
import { useSettingsFeedback } from "./settings-feedback";

type AutosaveOptions<T extends FieldValues> = {
  form: UseFormReturn<T>;
  /** Writes the values; resolves once they are kept. */
  save: (next: T, previous: T) => Promise<void>;
  /** Fields that save when they lose focus rather than on every keystroke. */
  textFields?: readonly string[];
  resourceName: string;
};

function snapshot<T extends FieldValues>(values: T): T {
  return structuredClone(values);
}

function sameValues<T extends FieldValues>(a: T, b: T): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}

function isTextField(name: string, textFields: readonly string[] | undefined): boolean {
  return Boolean(textFields?.some((field) => name === field || name.startsWith(`${field}.`)));
}

/**
 * Saves a settings form as it changes. A choice saves at once; a text field saves when it
 * loses focus. Each save offers Undo, which puts the previous values back and saves them.
 * A save that fails keeps what the person typed and shows why next to it.
 */
export function useAutosaveForm<T extends FieldValues>({
  form,
  save,
  textFields,
  resourceName,
}: AutosaveOptions<T>) {
  const feedback = useSettingsFeedback();
  const committed = useRef<T | null>(null);
  const saveRef = useRef(save);

  useLayoutEffect(() => {
    saveRef.current = save;
  });

  useEffect(() => {
    committed.current ??= snapshot(form.getValues());
  }, [form]);

  const persist = useCallback(
    async function persistValues(next: T, previous: T, offerUndo: boolean): Promise<boolean> {
      committed.current = next;
      try {
        await saveRef.current(next, previous);
      } catch (error) {
        committed.current = previous;
        handleMutationError({ error, form, resourceName });
        return false;
      }

      if (offerUndo) {
        feedback.saved(async () => {
          form.reset(previous, { keepDefaultValues: true });
          const restored = await persistValues(snapshot(previous), next, false);
          if (restored) {
            feedback.message(translate("Change undone"));
          }
        });
      } else {
        feedback.saved();
      }
      return true;
    },
    [feedback, form, resourceName],
  );

  const commit = useCallback(() => {
    const next = snapshot(form.getValues());
    const previous = committed.current ?? snapshot(form.formState.defaultValues as T);
    if (sameValues(next, previous)) {
      return;
    }
    form.clearErrors();
    void persist(next, previous, true);
  }, [form, persist]);

  const resync = useCallback(
    (values: T) => {
      committed.current = snapshot(values);
      form.reset(values);
    },
    [form],
  );

  useEffect(() => {
    const subscription = form.watch((_, { name, type }) => {
      if (type !== "change" || !name || isTextField(name, textFields)) {
        return;
      }
      commit();
    });
    return () => subscription.unsubscribe();
  }, [commit, form, textFields]);

  return { commit, resync };
}
