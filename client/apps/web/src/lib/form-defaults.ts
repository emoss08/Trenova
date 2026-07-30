import type { DefaultValues, FieldValues } from "react-hook-form";

type FormWithDefaults = {
  control: object;
  formState: { defaultValues?: unknown };
};

const pristineByControl = new WeakMap<object, unknown>();

// rememberPristineDefaults records the defaults a route gave its form, once, and hands them
// back on every later call.
//
// A route creates a single useForm and passes it to whichever of FormCreatePanel or
// FormEditPanel the current mode selects, so the two panels share one form instance.
// FormEditPanel loads a record with reset(row), and react-hook-form's reset REPLACES
// defaultValues with what it is given — which is what makes isDirty mean "changed since
// this record loaded" while editing. The cost is that a later bare reset() no longer
// restores a blank form; it restores the record that was last edited.
//
// Recording happens during render, before either panel's reset effect has run, so whichever
// panel mounts first captures the route's own defaults rather than a loaded record. Keyed on
// control because that object is stable for the life of the form and lets the entry be
// collected with it.
export function rememberPristineDefaults<TFieldValues extends FieldValues>(
  form: FormWithDefaults,
): DefaultValues<TFieldValues> | undefined {
  if (!pristineByControl.has(form.control)) {
    pristineByControl.set(form.control, form.formState.defaultValues);
  }

  return pristineByControl.get(form.control) as DefaultValues<TFieldValues> | undefined;
}
