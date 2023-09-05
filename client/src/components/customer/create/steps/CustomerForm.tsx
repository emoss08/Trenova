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
import { UseFormReturnType } from "@mantine/form";
import React from "react";
import { Box, SimpleGrid } from "@mantine/core";
import { TChoiceProps } from "@/types";
import { CreateCustomerFormValues } from "@/types/customer";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { statusChoices, yesAndNoChoices } from "@/helpers/constants";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { CityAutoCompleteField } from "@/components/common/fields/CityAutoCompleteField";
import { StateSelect } from "@/components/common/fields/StateSelect";

export function CreateCustomerModalForm({
  users,
  isUsersLoading,
  isUsersError,
  form,
}: {
  users: Array<TChoiceProps>;
  form: UseFormReturnType<CreateCustomerFormValues>;
  isUsersLoading: boolean;
  isUsersError: boolean;
}): React.ReactElement {
  const { classes } = useFormStyles();

  return (
    <Box className={classes.div}>
      <SimpleGrid
        cols={2}
        verticalSpacing="xs"
        spacing="lg"
        breakpoints={[{ maxWidth: "sm", cols: 1 }]}
      >
        <SelectInput<CreateCustomerFormValues>
          data={statusChoices}
          name="status"
          placeholder="Status"
          label="Status"
          description="Status of the customer"
          form={form}
          variant="filled"
          withAsterisk
        />
        <ValidatedTextInput<CreateCustomerFormValues>
          form={form}
          name="name"
          description="Name of the customer"
          label="Name"
          placeholder="Name"
          variant="filled"
          withAsterisk
        />
        <ValidatedTextInput<CreateCustomerFormValues>
          form={form}
          name="addressLine1"
          description="Address Line 1 of the customer"
          label="Address Line 1"
          placeholder="Address Line 1"
          variant="filled"
        />
        <ValidatedTextInput<CreateCustomerFormValues>
          form={form}
          description="Address Line 2 of the customer"
          name="addressLine2"
          label="Address Line 2"
          placeholder="Address Line 2"
          variant="filled"
        />
        <CityAutoCompleteField<CreateCustomerFormValues>
          form={form}
          stateSelection={form.values.city || ""}
          name="city"
          description="City of the customer"
          label="City"
          placeholder="City"
          variant="filled"
        />
        <StateSelect<CreateCustomerFormValues>
          label="State"
          placeholder="State"
          variant="filled"
          description="State of the customer"
          searchable
          form={form}
          name="state"
        />
        <ValidatedTextInput<CreateCustomerFormValues>
          form={form}
          name="zipCode"
          label="Zip Code"
          description="Zip Code of the customer"
          placeholder="Zip Code"
          variant="filled"
        />
        <SelectInput<CreateCustomerFormValues>
          data={yesAndNoChoices}
          name="hasCustomerPortal"
          placeholder="Has Customer Portal"
          label="Has Customer Portal"
          description="Customer has Customer Portal?"
          form={form}
          variant="filled"
          withAsterisk
        />
        <SelectInput<CreateCustomerFormValues>
          data={yesAndNoChoices}
          name="autoMarkReadyToBill"
          placeholder="Auto Mark Ready to Bill"
          label="Auto Mark Ready to Bill"
          description="Auto Mark Ready to Bill?"
          form={form}
          variant="filled"
          withAsterisk
        />
        <SelectInput<CreateCustomerFormValues>
          data={users}
          isLoading={isUsersLoading}
          isError={isUsersError}
          name="advocate"
          placeholder="Customer Advocate"
          label="Customer Advocate"
          description="Assigned Customer Advocate?"
          form={form}
          variant="filled"
        />
      </SimpleGrid>
    </Box>
  );
}
