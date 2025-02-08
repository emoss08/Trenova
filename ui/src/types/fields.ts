import type { InputProps } from "@/components/ui/input";
import type { TextareaProps } from "@/components/ui/textarea";
import { type IconDefinition } from "@fortawesome/pro-regular-svg-icons";
import { type CheckboxProps } from "@radix-ui/react-checkbox";
import type {
  Control,
  FieldValues,
  Path,
  RegisterOptions,
} from "react-hook-form";
import type { GroupBase, Props as ReactSelectProps } from "react-select";
import { type AsyncProps as ReactAsyncSelectProps } from "react-select/async";
import { type API_ENDPOINTS } from "./server";

type BaseInputFieldProps = Omit<InputProps, "name"> & {
  label?: string;
  description?: string;
  inputClassProps?: string;
};

export type InputFieldProps<T extends FieldValues> = BaseInputFieldProps &
  FormControlProps<T>;

type FormControlProps<T extends FieldValues> = {
  name: Path<T>;
  control: Control<T>;
  rules?: RegisterOptions<T, Path<T>>;
};

export type ColorFieldProps<TFieldValues extends FieldValues> = {
  description?: string;
  label?: string;
  className?: string;
} & FormControlProps<TFieldValues>;

type BaseCheckboxFieldProps = Omit<CheckboxProps, "name"> & {
  label: string;
  outlined?: boolean;
  description?: string;
};

export type CheckboxFieldProps<T extends FieldValues> = BaseCheckboxFieldProps &
  FormControlProps<T>;

type BaseTextareaFieldProps = Omit<TextareaProps, "name"> & {
  label: string;
  description?: string;
};

export type TextareaFieldProps<T extends FieldValues> = BaseTextareaFieldProps &
  FormControlProps<T>;

type BaseDateFieldProps = {
  label: string;
  description?: string;
  className?: string;
  onKeyDown?: (event: React.KeyboardEvent<HTMLInputElement>) => void;
  placeholder?: string;
};

export type DateFieldProps<T extends FieldValues> = BaseDateFieldProps &
  FormControlProps<T>;

export type SelectOption = {
  label: string;
  value: string | boolean | number;
  color?: string;
  description?: string;
  icon?: IconDefinition;
};

export type AddNewButtonProps = {
  label: string;
  popoutLink: string;
};

export type BaseSelectFieldProps = Omit<
  ReactSelectProps,
  "options" | "onChange" | "name"
> & {
  onChange: (...event: any[]) => void;
  options: SelectOption[];
  label?: string;
  description?: string;
  isReadOnly?: boolean;
  isInvalid?: boolean;
};

export type SelectFieldProps<T extends FieldValues> = BaseSelectFieldProps &
  FormControlProps<T>;

export type BaseAsyncSelectFieldProps = Omit<
  ReactAsyncSelectProps<SelectOption, boolean, GroupBase<SelectOption>>,
  "options" | "onChange" | "name"
> & {
  onChange: (...event: any[]) => void;
  link: API_ENDPOINTS;
  label?: string;
  description?: string;
  isReadOnly?: boolean;
  isInvalid?: boolean;
  isLoading?: boolean;
  isFetchError?: boolean;
  className?: string;
  valueKey?: string | string[];
  id?: string;
};

export type AsyncSelectFieldProps<T extends FieldValues> =
  BaseAsyncSelectFieldProps & FormControlProps<T>;

export type DoubleClickEditDateProps<T extends FieldValues> = {
  name: Path<T>;
  control: Control<T>;
  rules?: RegisterOptions<T, Path<T>>;
  placeholder?: string;
};

export type DoubleClickSelectFieldProps<T extends FieldValues> = {
  name: Path<T>;
  control: Control<T>;
  rules?: RegisterOptions<T, Path<T>>;
  placeholder?: string;
  options: SelectOption[];
};

export interface DatePickerProps
  extends React.InputHTMLAttributes<HTMLInputElement> {
  date: Date | undefined;
  setDate: (date: Date | undefined) => void;
  isInvalid?: boolean;
  placeholder?: string;
  clearable?: boolean;
  label?: string;
  description?: string;
}

export type AutoCompleteDateFieldProps<T extends FieldValues> = Omit<
  DatePickerProps,
  "date" | "setDate"
> &
  FormControlProps<T>;

export interface DateTimePickerProps
  extends React.InputHTMLAttributes<HTMLInputElement> {
  dateTime: Date | undefined;
  setDateTime: React.Dispatch<React.SetStateAction<Date | undefined>>;
}

export type Suggestion = {
  date: Date;
  inputString: string;
};
