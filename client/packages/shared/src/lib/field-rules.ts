/**
 * Whether a field must hold a value: `true`, or the words said under it when it does not.
 * Every shared form field takes it as its `required` prop. It is the one way a field is
 * marked required: the label's asterisk, the validation and whether the field can be
 * cleared all follow from it, so it is never also written into `rules`.
 */
export type FieldRequired = boolean | string;

/** A field's `rules` with its `required` prop folded in, as react-hook-form reads them. */
export function withRequired<TRules extends object>(
  rules: TRules | undefined,
  required: FieldRequired | undefined,
): TRules | undefined {
  return required === undefined ? rules : ({ ...rules, required } as TRules);
}

/** Whether a field is required, by its `required` prop or, failing that, its `rules`. */
export function isRequired(
  rules: { required?: unknown } | undefined,
  required: FieldRequired | undefined,
): boolean {
  return Boolean(required ?? rules?.required);
}
