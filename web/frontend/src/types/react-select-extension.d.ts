/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

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
