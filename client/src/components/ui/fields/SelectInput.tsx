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

import { Loader, Select, SelectItemProps, Text } from "@mantine/core";
import { IconAlertTriangle } from "@tabler/icons-react";
import React, { forwardRef } from "react";
import { SelectProps } from "@mantine/core/lib/Select/Select";
import { UseFormReturnType } from "@mantine/form";
import { useFormStyles } from "@/styles/FormStyles";

SelectInput.defaultProps = {
  isLoading: false,
  isError: false,
};

interface ValidatedSelectInputProps<TFormValues>
  extends Omit<SelectProps, "form" | "name"> {
  form: UseFormReturnType<TFormValues, (values: TFormValues) => TFormValues>;
  name: keyof TFormValues;
  isLoading?: boolean;
  isError?: boolean;
}

const SelectItem = forwardRef<HTMLDivElement, SelectItemProps>(
  ({ label, ...others }: SelectItemProps, ref) => (
    <div ref={ref} {...others}>
      <Text size="sm">{label}</Text>
    </div>
  ),
);

export function SelectInput<TFormValues extends Record<string, unknown>>({
  form,
  data = [],
  name,
  isLoading,
  isError,
  ...rest
}: ValidatedSelectInputProps<TFormValues>) {
  const { classes } = useFormStyles();
  const error = form.errors[name as string];

  const validatedData = Array.isArray(data) ? data : [];

  return (
    <Select
      {...rest}
      {...form.getInputProps(name as string)}
      data={validatedData}
      error={error}
      maxDropdownHeight={200}
      nothingFound="Nothing found"
      className={classes.fields}
      itemComponent={SelectItem}
      classNames={{
        input: error ? classes.invalid : "",
      }}
      filter={(value, item: any) =>
        item.label.toLowerCase().includes(value.toLowerCase().trim())
      }
      limit={10}
      disabled={isLoading}
      rightSection={
        isLoading ? (
          <Loader size={24} />
        ) : isError ? ( // Handle error state
          <IconAlertTriangle
            stroke={1.5}
            size="1.1rem"
            className={classes.invalidIcon}
          />
        ) : (
          error && (
            <IconAlertTriangle
              stroke={1.5}
              size="1.1rem"
              className={classes.invalidIcon}
            />
          )
        )
      }
      searchable
    />
  );
}
