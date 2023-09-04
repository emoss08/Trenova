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
import { IconAlertTriangle } from "@tabler/icons-react";
import { TextInput } from "@mantine/core";
import { TextInputProps } from "@mantine/core/lib/TextInput/TextInput";
import { UseFormReturnType } from "@mantine/form";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { InputFieldNameProp } from "@/types";

interface ValidatedTextInputProps<TFormValues>
  extends Omit<TextInputProps, "form" | "name"> {
  form: UseFormReturnType<TFormValues, (values: TFormValues) => TFormValues>;
  name: InputFieldNameProp<TFormValues>;
}

/**
 * A validated text input field that can be used with Mantine's form hooks.
 * @param form - The form hook to use.
 * @param name - The name of the field in the form.
 * @param rest - Any other props that can be passed to Mantine's TextInput component.
 */
export function ValidatedTextInput<TFormValues extends object>({
  form,
  name,
  ...rest
}: ValidatedTextInputProps<TFormValues>) {
  const { classes } = useFormStyles();
  const error = form.errors[name as string];

  return (
    <TextInput
      {...rest}
      {...form.getInputProps(name as string)}
      error={error}
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
