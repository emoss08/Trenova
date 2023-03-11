/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * Monta is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Monta is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with Monta.  If not, see <https://www.gnu.org/licenses/>.
 */

import React from "react";
import { components, DropdownIndicatorProps, GroupBase, StylesConfig } from "react-select";

import Select, { Props as SelectProps } from "react-select";
import AsyncSelect, {AsyncProps as AsyncSelectProps} from "react-select/async";

type StateSelectProps<O, M extends boolean, G extends GroupBase<O>> = {
  async?: boolean;
  components?: Partial<{
    Control: typeof components.Control;
    Option: typeof components.Option;
    MultiValue: typeof components.MultiValue;
    MultiValueLabel: typeof components.MultiValueLabel;
  } & {
    [key: string]: any;
  }>;
  size?: string;
} & (SelectProps<O> | AsyncSelectProps<O, M, G>);

function getSelectStyles(multi: boolean, size = ""): StylesConfig {
  const suffix = size ? `-${size}` : "";
  const multiplicator = multi ? 2 : 1;
  return {
    control: (provided, { isDisabled, isFocused }) => ({
      ...provided,
      color: 'var(--bs-select-color)',
      width: "100%",
      fontSize: `var(--bs-select-font-size${suffix})`,
      lineHeight: "var(--bs-select-line-height)",
      backgroundImage: "var(--bs-form-select-bg-img), var(--bs-form-select-bg-icon, none)",
      backgroundRepeat: "no-repeat",
      backgroundColor: `var(--bs-select${isDisabled ? "-disabled" : ""}-bg)`,
      border: "1px solid var(--bs-select-border-color)",
      transition: "border-color 0.15s ease-in-out, box-shadow 0.15s ease-in-out",
      transitionProperty: "border-color, box-shadow",
      transitionDuration: "0.15s, 0.15s",
      transitionTimingFunction: "ease-in-out, ease-in-out",
      transitionDelay: "0s, 0s",
      padding: "0.775rem 3rem, 0.775rem 1rem",
      borderRadius: "0.475rem",
      minHeight: `calc((var(--bs-select-line-height)*var(--bs-select-font-size${suffix})) + (var(--bs-select-padding-y${suffix})*2) + (var(--bs-select-border-width)*2))`,
      ":hover": {
        borderColor: "var(--bs-select-focus-border-color)"
      },
      ":focus": {
        borderColor: "var(--bs-select-focus-border-color)",
        boxShadow: "var(--bs-select-focus-box-shadow)"
      },
      ":focus-within": {
        borderColor: "var(--bs-select-focus-border-color)",
        boxShadow: "var(--bs-select-focus-box-shadow)"
      },
      ":active": {
        borderColor: "var(--bs-select-focus-border-color)",
        boxShadow: "var(--bs-select-focus-box-shadow)"
      }
    }),
    singleValue: ({ marginLeft, marginRight, ...provided }, { isDisabled }) => ({
      ...provided,
      color: `var(--bs-select${isDisabled ? "-disabled" : ""}-color)`
    }),
    valueContainer: (provided, state) => ({
      ...provided,
      padding: `calc(var(--bs-select-padding-y${suffix})/${multiplicator}) calc(var(--bs-select-padding-x${suffix})/${multiplicator})`
    }),
    dropdownIndicator: (provided, state) => ({
      height: "100%",
      width: "var(--bs-select-indicator-padding)",
      backgroundImage: "var(--bs-select-indicator)",
      backgroundRepeat: "no-repeat",
      backgroundPosition: `right var(--bs-select-padding-x) center`,
      backgroundSize: "var(--bs-select-bg-size)"
    }),
    input: ({ margin, paddingTop, paddingBottom, ...provided }, state) => ({
      ...provided,
      color: "var(--bs-select-color)"
    }),
    option: (provided, state) => ({
      ...provided,
      margin: `calc(var(--bs-select-padding-y${suffix})/2) calc(var(--bs-select-padding-x${suffix})/2)`,

    }),
    menu: ({ marginTop, ...provided }, state) => ({
      ...provided,
      backgroundColor: "var(--bs-select-bg)",
      borderColor: "var(--bs-select-border-color)",
      color: "var(--bs-select-color)"
    }),
    multiValue: (provided, state) => ({
      ...provided,
      margin: `calc(var(--bs-select-padding-y${suffix})/2) calc(var(--bs-select-padding-x${suffix})/2)`
    }),
    clearIndicator: ({ padding, ...provided }, state) => ({
      ...provided,
      alignItems: "center",
      justifyContent: "center",
      height: "100%",
      width: "var(--bs-select-indicator-padding)"
    }),
    multiValueLabel: ({ padding, paddingLeft, fontSize, ...provided }, state) => ({
      ...provided,
      padding: `0 var(--bs-select-padding-y${suffix})`,
      whiteSpace: "normal"
    })
  };
}

function IndicatorSeparator() {
  return null;
}

function DropdownIndicator(props: DropdownIndicatorProps) {
  return (
    <components.DropdownIndicator {...props}>
      <span></span>
    </components.DropdownIndicator>
  );
}

function getSelectTheme(theme: any) {
  return {
    ...theme,
    borderRadius: "var(--bs-select-border-radius)",
    colors: {
      ...theme.colors,
      primary: "var(--bs-primary)",
      danger: "var(--bs-danger)"
    }
  };
}

export default function StateSelect({ async, components, size, ...props }: StateSelectProps<any, boolean, any>) {
  const SelectType = async ? AsyncSelect : Select;
  return (
    <SelectType
      components={{ DropdownIndicator, IndicatorSeparator, ...components }}
      theme={getSelectTheme}
      styles={getSelectStyles("isMulti" in props, size)}
      {...props}
    />
  );
}