import type { FieldRequired } from "@trenova/shared/lib/field-rules";
import { type Select as SelectPrimitive } from "@base-ui/react/select";
import type React from "react";
import type { BadgeVariant } from "@trenova/shared/types/badge";
import type { Control, FieldValues, Path, RegisterOptions } from "react-hook-form";

export type FormControlProps<T extends FieldValues> = {
  name: Path<T>;
  control: Control<T>;
  /**
   * Whether the field must hold a value: `true`, or the words said when it does not. This
   * is how a field is marked required; it is not also written into `rules`.
   */
  required?: FieldRequired;
  /** Anything else react-hook-form should check, such as a `min` or a `validate`. */
  rules?: RegisterOptions<T, Path<T>>;
};

export type SelectOption = React.ComponentProps<typeof SelectPrimitive.Item> & {
  label: string;
  value: string | boolean | number | null;
  color?: string;
  description?: string;
  icon?: React.ReactNode;
  disabled?: boolean;
};

export type SelectOptionGroup = {
  label: string;
  options: SelectOption[];
};

export type GenericSelectOption<T extends string | boolean | number> = {
  value: T;
  label: string;
  color?: string;
  variant?: BadgeVariant;
  description?: string;
  icon?: React.ReactNode;
  disabled?: boolean;
};

export type WarningProps = {
  show: boolean;
  message: string;
};
