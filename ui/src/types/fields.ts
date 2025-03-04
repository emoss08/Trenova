import type { InputProps } from "@/components/ui/input";
import type { TextareaProps } from "@/components/ui/textarea";
import { type IconDefinition } from "@fortawesome/pro-regular-svg-icons";
import { type CheckboxProps } from "@radix-ui/react-checkbox";
import { SwitchProps } from "@radix-ui/react-switch";
import type {
  Control,
  FieldValues,
  Path,
  RegisterOptions,
} from "react-hook-form";
import type { Props as ReactSelectProps } from "react-select";
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
  disabled?: boolean;
} & FormControlProps<TFieldValues>;

type BaseCheckboxFieldProps = Omit<CheckboxProps, "name"> & {
  label: string;
  outlined?: boolean;
  description?: string;
};

export type CheckboxFieldProps<T extends FieldValues> = BaseCheckboxFieldProps &
  FormControlProps<T>;

type BaseSwitchFieldProps = Omit<SwitchProps, "name"> & {
  label: string;
  description?: string;
  outlined?: boolean;
};

export type SwitchFieldProps<T extends FieldValues> = BaseSwitchFieldProps &
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

export type Suggestion = {
  date: Date;
  inputString: string;
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
  setDateTime: (date: Date | undefined) => void;
  isInvalid?: boolean;
  placeholder?: string;
  clearable?: boolean;
  label?: string;
  description?: string;
  ref?: React.Ref<HTMLInputElement>;
}

export interface AutocompleteFormControlProps<T extends FieldValues> {
  name: Path<T>;
  control: Control<T>;
  rules?: RegisterOptions<T, Path<T>>;
}

export interface BaseAutocompleteFieldProps<TOption> {
  /** Link to fetch options */
  link: API_ENDPOINTS;
  /** Preload all data ahead of time */
  preload?: boolean;
  /** Function to filter options */
  filterFn?: (option: TOption, query: string) => boolean;
  /** Function to render each option */
  renderOption: (option: TOption) => React.ReactNode;
  /** Function to get the value from an option */
  getOptionValue: (option: TOption) => string | number;
  /** Function to get the display value for the selected option */
  getDisplayValue: (option: TOption) => React.ReactNode;
  /** Custom not found message */
  notFound?: React.ReactNode;
  /** Currently selected value */
  value: string;
  /** Callback when selection changes */
  onChange: (...event: any[]) => void;
  /** Label for the select field */
  label: string;
  /** Placeholder text when no selection */
  placeholder?: string;
  /** Disable the entire select */
  disabled?: boolean;
  /** Custom width for the popover */
  width?: string | number;
  /** Custom class names */
  className?: string;
  /** Custom trigger button class names */
  triggerClassName?: string;
  /** Custom no results message */
  noResultsMessage?: string;
  /** Allow clearing the selection */
  clearable?: boolean;
  /** Whether the field is invalid */
  isInvalid?: boolean;
  /** Callback when an option is selected (Specific to AutocompleteField) */
  onOptionChange?: (option: TOption | null) => void;
  /** Extra search params to append to the query */
  extraSearchParams?: Record<string, string>;
}

export type AutocompleteFieldProps<TOption, TForm extends FieldValues> = Omit<
  BaseAutocompleteFieldProps<TOption>,
  "onChange" | "value"
> &
  AutocompleteFormControlProps<TForm> & {
    description?: string;
  };
