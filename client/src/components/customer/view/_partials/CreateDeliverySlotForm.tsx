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
import { UseFormReturnType } from "@mantine/form";
import { Box, Button, Divider, Group, useMantineTheme } from "@mantine/core";
import { ValidatedTimeInput } from "@/components/common/fields/TimeInput";
import { CreateCustomerFormValues } from "@/types/customer";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { DayOfWeekChoices } from "@/helpers/choices";
import { TChoiceProps } from "@/types";

export function DeliverySlotForm({
  locations,
  isLocationsLoading,
  isLocationsError,
  form,
}: {
  locations: Array<TChoiceProps>;
  form: UseFormReturnType<CreateCustomerFormValues>;
  isLocationsLoading: boolean;
  isLocationsError: boolean;
}) {
  const theme = useMantineTheme();

  const fields = form.values.deliverySlots?.map((item, index) => (
    <>
      <Group mt="xs" key={index}>
        <ValidatedTimeInput<CreateCustomerFormValues>
          label="Start Time"
          form={form}
          name={`deliverySlots.${index}.startTime`}
          sx={{ flex: 1 }}
          placeholder="Enter Start Time"
          variant="filled"
          withSeconds
        />
        <ValidatedTimeInput<CreateCustomerFormValues>
          label="End Time"
          form={form}
          name={`deliverySlots.${index}.endTime`}
          sx={{ flex: 1 }}
          placeholder="Enter End Time"
          variant="filled"
          withSeconds
        />
        <SelectInput<CreateCustomerFormValues>
          form={form}
          label="Day of Week"
          name={`deliverySlots.${index}.dayOfWeek`}
          data={DayOfWeekChoices}
          sx={{ flex: 1 }}
          placeholder="Select Day of Week"
          variant="filled"
        />
      </Group>
      <Group spacing="xl">
        <SelectInput<CreateCustomerFormValues>
          form={form}
          name={`deliverySlots.${index}.location`}
          data={locations}
          isLoading={isLocationsLoading}
          isError={isLocationsError}
          label="Location"
          placeholder="Select Location"
          variant="filled"
        />
        <Button
          mt={40}
          variant="subtle"
          style={{
            color:
              theme.colorScheme === "dark"
                ? theme.colors.gray[0]
                : theme.colors.dark[9],
            backgroundColor: "transparent",
          }}
          size="sm"
          compact
          onClick={() => form.removeListItem("deliverySlots", index)}
        >
          Remove Delivery Slot
        </Button>
      </Group>
      <Divider variant="dashed" mt={20} />
    </>
  ));

  return (
    <Box>
      {fields}
      <Button
        variant="subtle"
        style={{
          color:
            theme.colorScheme === "dark"
              ? theme.colors.gray[0]
              : theme.colors.dark[9],
          backgroundColor: "transparent",
        }}
        size="sm"
        compact
        mt={20}
        onClick={() =>
          form.insertListItem("deliverySlots", {
            dayOfWeek: "MON",
            startTime: "",
            endTime: "",
            location: "",
          })
        }
      >
        Add Delivery Slot
      </Button>
    </Box>
  );
}
