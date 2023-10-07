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
import { Box, Divider, SimpleGrid, Text } from "@mantine/core";
import { TChoiceProps } from "@/types";
import { RateFormValues as FormValues } from "@/types/dispatch";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { getNewRateNumber } from "@/services/DispatchRequestService";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { rateMethodChoices, statusChoices } from "@/lib/constants";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedDateInput } from "@/components/common/fields/DateInput";
import { ValidatedNumberInput } from "@/components/common/fields/NumberInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";

type Props = {
  customers: ReadonlyArray<TChoiceProps>;
  isCustomersLoading: boolean;
  isCustomersError: boolean;
  commodities: ReadonlyArray<TChoiceProps>;
  isCommoditiesLoading: boolean;
  isCommoditiesError: boolean;
  orderTypes: ReadonlyArray<TChoiceProps>;
  isOrderTypesLoading: boolean;
  isOrderTypesError: boolean;
  equipmentTypes: ReadonlyArray<TChoiceProps>;
  isEquipmentTypesLoading: boolean;
  isEquipmentTypesError: boolean;
  locations: ReadonlyArray<TChoiceProps>;
  isLocationsLoading: boolean;
  isLocationsError: boolean;
  form: UseFormReturnType<FormValues>;
};

export default function RateForm({
  customers,
  isCustomersLoading,
  isCustomersError,
  commodities,
  isCommoditiesLoading,
  isCommoditiesError,
  orderTypes,
  isOrderTypesLoading,
  isOrderTypesError,
  equipmentTypes,
  isEquipmentTypesLoading,
  isEquipmentTypesError,
  locations,
  isLocationsLoading,
  isLocationsError,
  form,
}: Props) {
  const { classes } = useFormStyles();

  const fetchRate = React.useCallback(async () => {
    try {
      const rateNumber = await getNewRateNumber();
      form.setFieldValue("rateNumber", rateNumber);
    } catch (err) {
      console.error("Error fetching rate number:", err);
    }
  }, []);

  // fetch rate number and assign it to rateNumber field once when component mounts
  React.useEffect(() => {
    fetchRate();
  }, [fetchRate]);

  return (
    <Box className={classes.div}>
      <SimpleGrid cols={4} breakpoints={[{ maxWidth: "lg", cols: 1 }]}>
        <SelectInput<FormValues>
          form={form}
          data={statusChoices}
          description="Status of Rate"
          name="status"
          label="Status"
          placeholder="Status"
          variant="filled"
          withAsterisk
        />
        <ValidatedTextInput<FormValues>
          form={form}
          name="rateNumber"
          label="Rate Number"
          placeholder="Rate Number"
          description="Unique Number for Rate"
          disabled
          withAsterisk
        />
      </SimpleGrid>
      <Box my={10}>
        {" "}
        {/** Move into */}
        <div className="flex flex-col items-center justify-center text-center">
          <Text fw={400} fz="lg" className={classes.text}>
            Rate Details
          </Text>
        </div>
        <Divider my={5} variant="dashed" />
      </Box>
      <SimpleGrid cols={4} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
        <SelectInput<FormValues>
          form={form}
          name="customer"
          label="Customer"
          placeholder="Customer"
          description="Customer associated with this Rate"
          data={customers} // TODO(WOLFRED): add context menu's or add creatable to select fields
          isLoading={isCustomersLoading}
          isError={isCustomersError}
        />
        <SelectInput<FormValues>
          form={form}
          name="commodity"
          label="Commodity"
          placeholder="Commodity"
          description="Commodity associated with this Rate"
          data={commodities}
          isLoading={isCommoditiesLoading}
          isError={isCommoditiesError}
        />
        <ValidatedDateInput<FormValues>
          form={form}
          name="effectiveDate"
          label="Effective Date"
          placeholder="Effective Date"
          description="Effective Date for Rate"
          withAsterisk
        />
        <ValidatedDateInput<FormValues>
          form={form}
          name="expirationDate"
          label="Expiration Date"
          placeholder="Expiration Date"
          description="Expiration Date for Rate"
          withAsterisk
        />
        <SelectInput<FormValues>
          form={form}
          name="originLocation"
          label="Origin Location"
          placeholder="Origin Location"
          description="Origin Location associated with this Rate"
          data={locations}
          isLoading={isLocationsLoading}
          isError={isLocationsError}
        />
        <SelectInput<FormValues>
          form={form}
          name="destinationLocation"
          label="Destination Location"
          placeholder="Destination Location"
          description="Dest. Location associated with this Rate"
          data={locations}
          isLoading={isLocationsLoading}
          isError={isLocationsError}
        />
        <SelectInput<FormValues>
          form={form}
          name="equipmentType"
          label="Equipment Type"
          placeholder="Equipment Type"
          description="Equipment Type associated with this Rate"
          data={equipmentTypes}
          isLoading={isEquipmentTypesLoading}
          isError={isEquipmentTypesError}
        />
        <SelectInput<FormValues>
          form={form}
          name="orderType"
          label="Order Type"
          placeholder="Order Type"
          description="Order Type associated with this Rate"
          data={orderTypes}
          isLoading={isOrderTypesLoading}
          isError={isOrderTypesError}
        />
        <SelectInput<FormValues>
          form={form}
          name="rateMethod"
          label="Rate Method"
          placeholder="Rate Method"
          description="Rate Method associated with this Rate"
          data={rateMethodChoices}
          withAsterisk
        />
        <ValidatedNumberInput<FormValues>
          form={form}
          name="rateAmount"
          label="Rate Amount"
          placeholder="Rate Amount"
          description="Rate Amount associated with this Rate"
          withAsterisk
        />
        <ValidatedNumberInput<FormValues>
          form={form}
          name="distanceOverride"
          label="Distance Override"
          placeholder="Distance Override"
          description="Dist. Override associated with this Rate"
          withAsterisk
        />
      </SimpleGrid>
      <ValidatedTextArea<FormValues>
        form={form}
        name="comments"
        label="Comments"
        description="Additional Comments for Rate"
        placeholder="Comments"
      />
    </Box>
  );
}
