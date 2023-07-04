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
import { SwitchInput } from "@/components/ui/fields/SwitchInput";
import { useMutation, useQueryClient } from "react-query";
import axios from "@/lib/AxiosConfig";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { APIError } from "@/types/server";
import { OrderControl, OrderControlFormValues } from "@/types/apps/order";
import { orderControlSchema } from "@/utils/apps/order/schema";
import { useFormStyles } from "@/styles/FormStyles";

interface Props {
  orderControl: OrderControl;
}

export const OrderControlForm: React.FC<Props> = ({ orderControl }) => {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: OrderControlFormValues) =>
      axios.put(`/order_control/${orderControl.id}/`, values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["orderControl"],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "Order Control updated successfully",
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

  const form = useForm<OrderControlFormValues>({
    validate: yupResolver(orderControlSchema),
    initialValues: {
      auto_rate_orders: orderControl.auto_rate_orders,
      calculate_distance: orderControl.calculate_distance,
      enforce_rev_code: orderControl.enforce_rev_code,
      enforce_voided_comm: orderControl.enforce_voided_comm,
      generate_routes: orderControl.generate_routes,
      enforce_commodity: orderControl.enforce_commodity,
      auto_sequence_stops: orderControl.auto_sequence_stops,
      auto_order_total: orderControl.auto_order_total,
      enforce_origin_destination: orderControl.enforce_origin_destination,
      check_for_duplicate_bol: orderControl.check_for_duplicate_bol,
      remove_orders: orderControl.remove_orders,
    },
  });

  const handleSubmit = (values: OrderControlFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => handleSubmit(values))}>
      <Box className={classes.div}>
        <Box>
          <SimpleGrid cols={3} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <SwitchInput
              form={form}
              className={classes.fields}
              name="auto_rate_orders"
              label="Auto Rate Orders"
              description="Automatically rate orders when they are created"
            />
            <SwitchInput
              form={form}
              className={classes.fields}
              name="calculate_distance"
              label="Auto Calculate Distance"
              description="Automatically Calculate distance between stops"
            />
            <SwitchInput
              form={form}
              className={classes.fields}
              name="enforce_rev_code"
              label="Enforce Rev Code"
              description="Enforce rev code code when entering an order."
            />
            <SwitchInput
              form={form}
              className={classes.fields}
              name="generate_routes"
              label="Auto Generate Routes"
              description="Automatically generate routing information for the order."
            />
            <SwitchInput
              form={form}
              className={classes.fields}
              name="enforce_commodity"
              label="Enforce Commodity"
              description="Enforce the commodity input on the entry of an order."
            />
            <SwitchInput
              form={form}
              className={classes.fields}
              name="auto_sequence_stops"
              label="Auto Sequence Stops"
              description="Auto Sequence stops for the order and movements."
            />
            <SwitchInput
              form={form}
              className={classes.fields}
              name="auto_order_total"
              label="Auto Total Orders"
              description="Automate the order total amount calculation."
            />
            <SwitchInput
              form={form}
              className={classes.fields}
              name="enforce_origin_destination"
              label="Enforce Origin Destination"
              description="Compare and validate that origin and destination are not the same."
            />
            <SwitchInput
              form={form}
              className={classes.fields}
              name="check_for_duplicate_bol"
              label="Check for Duplicate BOL"
              description="Check for duplicate BOL numbers when entering an order."
            />
            <SwitchInput
              form={form}
              className={classes.fields}
              name="remove_orders"
              label="Allow Order Removal"
              description="Ability to remove orders from system. This will disallow the removal of Orders, Movements and Stops."
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
