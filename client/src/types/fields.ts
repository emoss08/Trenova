import type { badgeVariants } from "@/components/ui/badge";
import { type Select as SelectPrimitive } from "@base-ui/react/select";
import type { VariantProps } from "class-variance-authority";
import type React from "react";
import type { Control, FieldValues, Path, RegisterOptions } from "react-hook-form";

export type FormControlProps<T extends FieldValues> = {
  name: Path<T>;
  control: Control<T>;
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
  variant?: VariantProps<typeof badgeVariants>["variant"];
  description?: string;
  icon?: React.ReactNode;
  disabled?: boolean;
};

export type WarningProps = {
  show: boolean
  message: string
}