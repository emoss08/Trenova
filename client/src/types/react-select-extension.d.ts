import { CreatableProps, Props } from "react-select/creatable";

declare module "react-select/dist/declarations/src/Select" {
  export interface Props<
    Option,
    IsMulti extends boolean,
    Group extends GroupBase<Option>,
  > {
    isFetchError?: boolean;
    formError?: string;
    isLoading?: boolean;
    maxOptions?: number;
    isDisabled?: boolean;
    isClearable?: boolean;
    isMulti?: IsMulti;
    placeholder?: string;
  }
  export interface CreatableProps<
    Option,
    IsMulti extends boolean,
    Group extends GroupBase<Option>,
  > {
    isFetchError?: boolean;
    formError?: string;
  }
}

// override the default Props on GroupBase

declare module "react-select/dist/declarations/src/types" {
  export interface GroupBase<Option> {
    label: string;
    options: Options<Option>;
    value: string;
  }
}
