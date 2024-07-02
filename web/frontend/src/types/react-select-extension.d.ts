import type { } from "react-select/base";

import { IconProp } from "@fortawesome/fontawesome-svg-core"; // eslint-disable-line import/no-unassigned-import

declare module "react-select/base" {
  export interface Props<
    Option,
    IsMulti extends boolean,
    Group extends GroupBase<Option>,
  > {
    isFetchError?: boolean;
    formError?: string;
    isLoading?: boolean;
    maxOptions?: number;
    hasPopoutWindow?: boolean; // Flag when to show the add new option
    popoutLink?: string; // Link to the popout page
    popoutLinkLabel?: string; // Label for the popout link
    isDisabled?: boolean;
    isClearable?: boolean;
    isMulti?: IsMulti;
    placeholder?: string;
    Group?: Group;
  }
  export interface CreatableProps<
    Option,
    IsMulti extends boolean,
    Group extends GroupBase<Option>,
  > {
    isFetchError?: boolean;
    formError?: string;
    isMulti?: IsMulti;
    hasPopoutWindow?: boolean; // Flag when to show the add new option
    popoutLink?: string; // Link to the popout page
    popoutLinkLabel?: string; // Label for the popout link
    Group?: Group;
  }
}

declare module "react-select/async/dist" {
  export interface Props<
    Option,
    IsMulti extends boolean,
    Group extends GroupBase<Option>,
  > {
    isFetchError?: boolean;
    formError?: string;
    maxOptions?: number;
    hasPopoutWindow?: boolean; // Flag when to show the add new option
    popoutLink?: string; // Link to the popout page
    popoutLinkLabel?: string; // Label for the popout link
    isDisabled?: boolean;
    isClearable?: boolean;
    isMulti?: IsMulti;
    placeholder?: string;
    Group?: Group;
  }
}

// override the default Props on GroupBase
declare module "react-select/dist/declarations/src/types" {
  export interface GroupBase<Option> {
    options: Options<Option>;
    value: string;
  }
}

declare module "react-select/dist/declarations/src/components/Option" {
  export interface OptionProps {
    data: {
      value: string;
      label: string;
      icon?: IconProp;
      color?: string;
      description?: string;
    };
  }
}
