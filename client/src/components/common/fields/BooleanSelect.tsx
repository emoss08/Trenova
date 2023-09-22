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

import { Select, SelectItemProps, Text } from "@mantine/core";
import React, { forwardRef } from "react";
import { SelectProps } from "@mantine/core/lib/Select/Select";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { BChoiceProps } from "@/types";

interface BooleanSelectInputProps
  extends Omit<SelectProps, "form" | "data" | "value"> {
  data: ReadonlyArray<BChoiceProps>;
  value: boolean;
}

const SelectItem = forwardRef<HTMLDivElement, SelectItemProps>(
  ({ label, ...others }: SelectItemProps, ref) => (
    <div ref={ref} {...others}>
      <Text size="sm">{label}</Text>
    </div>
  ),
);

export function BooleanSelectInput({
  data = [],
  value,
  ...rest
}: BooleanSelectInputProps): React.ReactElement {
  const { classes } = useFormStyles();
  const validatedData = Array.isArray(data) ? data : [];

  return (
    <Select
      {...rest}
      data={validatedData}
      maxDropdownHeight={200}
      nothingFound="Nothing found"
      className={classes.fields}
      itemComponent={SelectItem}
      // @ts-ignore
      value={value}
      limit={10}
      variant="filled"
      searchable
    />
  );
}
