/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import type { } from "react-select/base";

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
