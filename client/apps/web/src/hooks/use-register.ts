import type {
  Control,
  FieldPath,
  FieldValues,
  RegisterOptions,
  UseFormRegisterReturn,
} from "react-hook-form";

/*
 * react-hook-form expects register() to run on every render: reset() forgets every
 * registered field, and the next render's register() calls bring them back. The React
 * Compiler memoizes a `{...register(name)}` spread with the form, which never changes,
 * so after a reset an input that stayed on screen is never registered again and what is
 * typed into it no longer reaches the form. This hook opts out of memoization, so the
 * field is registered on every render and its props are new each time.
 */

/** Registers an uncontrolled input with the form, every render, and returns its props. */
export function useRegister<T extends FieldValues, N extends FieldPath<T>>(
  control: Control<T, any, any>,
  name: N,
  options?: RegisterOptions<T, N>,
): UseFormRegisterReturn<N> {
  "use no memo";
  return control.register(name, options);
}
