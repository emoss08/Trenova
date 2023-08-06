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

import { Box, Button, Group, SimpleGrid } from "@mantine/core";
import React from "react";
import { useForm, yupResolver } from "@mantine/form";
import { useMutation, useQueryClient } from "react-query";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import axios from "@/lib/AxiosConfig";
import { SwitchInput } from "@/components/ui/fields/SwitchInput";
import { SelectInput } from "@/components/ui/fields/SelectInput";
import { APIError } from "@/types/server";
import {
  DispatchControl,
  DispatchControlFormValues,
} from "@/types/apps/dispatch";
import { serviceIncidentControlChoices } from "@/utils/apps/dispatch";
import { ValidatedNumberInput } from "@/components/ui/fields/NumberInput";
import { ValidatedTextInput } from "@/components/ui/fields/TextInput";
import { dispatchControlSchema } from "@/utils/apps/dispatch/schema";
import { useFormStyles } from "@/styles/FormStyles";

interface Props {
  dispatchControl: DispatchControl;
}

export const DispatchControlForm: React.FC<Props> = ({ dispatchControl }) => {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: DispatchControlFormValues) =>
      axios.put(`/dispatch_control/${dispatchControl.id}/`, values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["dispatchControl"],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "Dispatch Control updated successfully",
              color: "green",
              withCloseButton: true,
              icon: <FontAwesomeIcon icon={faCheck} />,
            });
          });
      },
      onError: (error: any) => {
        const { data } = error.response;
        if (data.type === "validation_error") {
          data.errors.forEach((error: APIError) => {
            form.setFieldError(error.attr, error.detail);
            if (error.attr === "non_field_errors") {
              notifications.show({
                title: "Error",
                message: error.detail,
                color: "red",
                withCloseButton: true,
                icon: <FontAwesomeIcon icon={faXmark} />,
                autoClose: 10_000, // 10 seconds
              });
            }
          });
        }
      },
      onSettled: () => {
        setLoading(false);
      },
    }
  );

  const form = useForm<DispatchControlFormValues>({
    validate: yupResolver(dispatchControlSchema),
    initialValues: {
      record_service_incident: dispatchControl.record_service_incident,
      grace_period: dispatchControl.grace_period,
      deadhead_target: dispatchControl.deadhead_target,
      driver_assign: dispatchControl.driver_assign,
      trailer_continuity: dispatchControl.trailer_continuity,
      dupe_trailer_check: dispatchControl.dupe_trailer_check,
      regulatory_check: dispatchControl.regulatory_check,
      prev_orders_on_hold: dispatchControl.prev_orders_on_hold,
      driver_time_away_restriction:
        dispatchControl.driver_time_away_restriction,
      tractor_worker_fleet_constraint:
        dispatchControl.tractor_worker_fleet_constraint,
    },
  });

  const handleSubmit = (values: DispatchControlFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => handleSubmit(values))}>
      <Box className={classes.div}>
        <Box>
          <SimpleGrid cols={3} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <SelectInput
              form={form}
              data={serviceIncidentControlChoices}
              className={classes.fields}
              name="record_service_incident"
              label="Record Service Incident"
              placeholder="Record Service Incident"
              description="Record service incident for the company."
              variant="filled"
              withAsterisk
            />
            <ValidatedNumberInput
              form={form}
              className={classes.fields}
              name="grace_period"
              label="Grace Period"
              placeholder="Grace Period"
              description="Grace period for the service incident in minutes."
              variant="filled"
              withAsterisk
            />
            <ValidatedTextInput
              form={form}
              className={classes.fields}
              name="deadhead_target"
              label="Deadhead Target"
              placeholder="Deadhead Target"
              description="Deadhead target for the company."
              variant="filled"
              withAsterisk
            />
            <SwitchInput
              form={form}
              className={classes.fields}
              name="driver_assign"
              label="Driver Assign"
              description="Enforce driver assign to orders for the company."
            />
            <SwitchInput
              form={form}
              className={classes.fields}
              name="trailer_continuity"
              label="Trailer Continuity"
              description="Enforce trailer continuity for the company."
            />
            <SwitchInput
              form={form}
              className={classes.fields}
              name="dupe_trailer_check"
              label="Dupe Trailer Check"
              description="Enforce duplicate trailer check for the company."
            />
            <SwitchInput
              form={form}
              className={classes.fields}
              name="regulatory_check"
              label="Regulatory Check"
              description="Enforce regulatory check for the company."
            />
            <SwitchInput
              form={form}
              className={classes.fields}
              name="prev_orders_on_hold"
              label="Previous Orders on Hold"
              description="Prevent dispatch of orders on hold for the company."
            />
            <SwitchInput
              form={form}
              className={classes.fields}
              name="driver_time_away_restriction"
              label="Driver Time Away Restriction"
              description="Disallow assignments if the driver is on Time Away."
            />
            <SwitchInput
              form={form}
              className={classes.fields}
              name="tractor_worker_fleet_constraint"
              label="Tractor Worker Fleet Constraint"
              description="Enforce Worker and Tractor must be in the same fleet to be assigned to a dispatch."
            />
          </SimpleGrid>
          <Group position="right" mt="md">
            <Button
              color="white"
              type="submit"
              className={classes.control}
              loading={loading}
            >
              Submit
            </Button>
          </Group>
        </Box>
      </Box>
    </form>
  );
};
