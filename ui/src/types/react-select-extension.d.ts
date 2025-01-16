import type { } from "react-select/base";

// Common props shared across different interfaces
interface CommonProps {
  isFetchError?: boolean;
  formError?: string;
  isLoading?: boolean;
  maxOptions?: number;
  hasPopoutWindow?: boolean;
  hasPermission?: boolean;
  popoutLink?: string;
  popoutLinkLabel?: string;
  isDisabled?: boolean;
  isClearable?: boolean;
  placeholder?: string;
}

// Extended option type
export interface ExtendedOption {
  value: string;
  label: string;
  icon?: IconDefinition;
  color?: string;
  description?: string;
}

declare module "react-select" {
  export interface GroupBase<Option> {
    options: Option[];
    label: string;
  }
}

declare module "react-select/base" {
  export interface Props<
    Option,
    IsMulti extends boolean,
    Group extends GroupBase<Option>,
  > extends BaseProps<Option, IsMulti, Group>,
      CommonProps {
    isMulti?: IsMulti;
    popoutLinkLabel?: string;
    Group?: Group;
  }
}

// declare module "react-select/async" {
//   export interface AsyncProps<
//     Option,
//     IsMulti extends boolean,
//     Group extends GroupBase<Option>,
//   > extends BaseAsyncProps<Option, IsMulti, Group>,
//       CommonProps {
//     isMulti?: IsMulti;
//     Group?: Group;
//     addPermission: string;
//     label?: string;
//     description?: string;
//     isReadOnly?: boolean;
//     link?: API_ENDPOINTS;
//     menuPlacement?: MenuPlacement;
//     menuPosition?: MenuPosition;
//     hideSelectedOptions?: boolean;
//     hasPopoutWindow?: boolean;
//     popoutLink?: string;
//     popoutLinkLabel?: string;
//   }
// }

declare module "react-select/creatable" {
  export interface CreatableProps<
    Option,
    IsMulti extends boolean,
    Group extends GroupBase<Option>,
  > extends BaseCreatableProps<Option, IsMulti, Group>,
      CommonProps {
    isMulti?: IsMulti;
    Group?: Group;
  }
}

declare module "react-select/dist/declarations/src/types" {
  export interface GroupBase<Option> {
    options: ReadonlyArray<Option>;
    value: string;
  }

  export interface SelectProps<
    Option,
    IsMulti extends boolean,
    Group extends GroupBase<Option>,
  > {
    popoutLinkLabel?: string;
  }
}

declare module "react-select/dist/declarations/src/components/Option" {
  export interface OptionProps<
    Option = unknown,
    IsMulti extends boolean = boolean,
    Group extends GroupBase<Option> = GroupBase<Option>,
  > {
    data: ExtendedOption;
    selectProps: Props<Option, IsMulti, Group>;
  }
}

declare module "react-select/dist/declarations/src/components/Menu" {
  export interface MenuListProps<
    Option,
    IsMulti extends boolean,
    Group extends GroupBase<Option>,
  > {
    selectProps: Props<Option, IsMulti, Group>;
  }
}

declare module "react-select/dist/declarations/src/components/SingleValue" {
  export interface SingleValueProps<
    Option,
    IsMulti extends boolean,
    Group extends GroupBase<Option>,
  > {
    data: Option;
    selectProps: Props<Option, IsMulti, Group>;
  }
}
