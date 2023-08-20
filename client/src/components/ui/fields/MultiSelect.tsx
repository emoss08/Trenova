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
import React from "react";
import {
  Loader,
  MultiSelect,
  MultiSelectProps,
  rem,
  useMantineTheme,
} from "@mantine/core";
import { UseFormReturnType } from "@mantine/form";
import { IconAlertTriangle } from "@tabler/icons-react";
import { useFormStyles } from "@/styles/FormStyles";

ValidatedMultiSelect.defaultProps = {
  isLoading: false,
};

interface ValidatedMultiSelectProps<TFormValues>
  extends Omit<MultiSelectProps, "form"> {
  form: UseFormReturnType<TFormValues, (values: TFormValues) => TFormValues>;
  isLoading?: boolean;
}

export function ValidatedMultiSelect<TFormValues extends object>({
  form,
  name,
  isLoading,
  ...rest
}: ValidatedMultiSelectProps<TFormValues>): React.ReactElement {
  const { classes } = useFormStyles();
  const error: React.ReactNode = form.errors[name as string];
  const theme = useMantineTheme();

  return (
    <MultiSelect
      {...rest}
      {...form.getInputProps(name as string)}
      error={error}
      styles={{
        label: {
          marginTop: rem(10),
        },
        input: {
          backgroundColor:
            theme.colorScheme === "dark"
              ? theme.colors.dark[6]
              : theme.colors.gray[1],
          "& [data-invalid=true]": {
            borderColor: theme.colors.red[6],
          },
        },
      }}
      disabled={isLoading}
      rightSection={
        isLoading ? (
          <Loader size={24} />
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
      nothingFound="Nothing found"
    />
  );
}
