/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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
import React, { forwardRef } from "react";
import { Text, Select, SelectProps } from "@mantine/core";
import { IconAlertTriangle } from "@tabler/icons-react";
import { UseFormReturnType } from "@mantine/form";
import { TChoiceProps } from "@/types";
import { useFormStyles } from "@/styles/FormStyles";

export const stateData = [
  { label: "Alabama", value: "AL" },
  { label: "Alaska", value: "AK" },
  { label: "Arizona", value: "AZ" },
  { label: "Arkansas", value: "AR" },
  { label: "California", value: "CA" },
  { label: "Colorado", value: "CO" },
  { label: "Connecticut", value: "CT" },
  { label: "Delaware", value: "DE" },
  { label: "Florida", value: "FL" },
  { label: "Georgia", value: "GA" },
  { label: "Hawaii", value: "HI" },
  { label: "Idaho", value: "ID" },
  { label: "Illinois", value: "IL" },
  { label: "Indiana", value: "IN" },
  { label: "Iowa", value: "IA" },
  { label: "Kansas", value: "KS" },
  { label: "Kentucky", value: "KY" },
  { label: "Louisiana", value: "LA" },
  { label: "Maine", value: "ME" },
  { label: "Maryland", value: "MD" },
  { label: "Massachusetts", value: "MA" },
  { label: "Michigan", value: "MI" },
  { label: "Minnesota", value: "MN" },
  { label: "Mississippi", value: "MS" },
  { label: "Missouri", value: "MO" },
  { label: "Montana", value: "MT" },
  { label: "Nebraska", value: "NE" },
  { label: "Nevada", value: "NV" },
  { label: "New Hampshire", value: "NH" },
  { label: "New Jersey", value: "NJ" },
  { label: "New Mexico", value: "NM" },
  { label: "New York", value: "NY" },
  { label: "North Carolina", value: "NC" },
  { label: "North Dakota", value: "ND" },
  { label: "Ohio", value: "OH" },
  { label: "Oklahoma", value: "OK" },
  { label: "Oregon", value: "OR" },
  { label: "Pennsylvania", value: "PA" },
  { label: "Rhode Island", value: "RI" },
  { label: "South Carolina", value: "SC" },
  { label: "South Dakota", value: "SD" },
  { label: "Tennessee", value: "TN" },
  { label: "Texas", value: "TX" },
  { label: "Utah", value: "UT" },
  { label: "Vermont", value: "VT" },
  { label: "Virginia", value: "VA" },
  { label: "Washington", value: "WA" },
  { label: "West Virginia", value: "WV" },
  { label: "Wisconsin", value: "WI" },
  { label: "Wyoming", value: "WY" },
] satisfies ReadonlyArray<TChoiceProps>;

interface StateSelectProps<TFormValues extends object>
  extends Omit<SelectProps, "data" | "form"> {
  form: UseFormReturnType<TFormValues, (values: TFormValues) => TFormValues>;
  name: string;
  searchable: boolean;
}

interface ItemProps extends React.ComponentPropsWithoutRef<"div"> {
  label: string;
  value: string;
  image: string;
}

const SelectItem = forwardRef<HTMLDivElement, ItemProps>(
  ({ label, ...others }: ItemProps, ref) => (
    <div ref={ref} {...others}>
      <Text size="sm">{label}</Text>
    </div>
  ),
);

export function StateSelect<StateFormValues extends object>({
  form,
  name,
  searchable,
  ...rest
}: StateSelectProps<StateFormValues>) {
  const { classes } = useFormStyles();
  const error = form.errors[name as string];

  return (
    <Select
      {...rest}
      itemComponent={SelectItem}
      data={stateData}
      maxDropdownHeight={200}
      nothingFound="Nothing found"
      filter={(value, item: any) =>
        item.label.toLowerCase().includes(value.toLowerCase().trim())
      }
      {...form.getInputProps(name)}
      error={error}
      searchable={searchable}
      className={classes.fields}
      rightSection={
        error && (
          <IconAlertTriangle
            stroke={1.5}
            size="1.1rem"
            className={classes.invalidIcon}
          />
        )
      }
    />
  );
}
