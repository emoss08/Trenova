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

import { useMutation, useQuery, useQueryClient } from "react-query";
import {
  Box,
  Button,
  Card,
  Divider,
  Group,
  SimpleGrid,
  Skeleton,
  Text,
} from "@mantine/core";
import React from "react";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useForm, yupResolver } from "@mantine/form";
import { getOrderControl } from "@/services/OrganizationRequestService";
import { usePageStyles } from "@/assets/styles/PageStyles";
import { OrderControl, OrderControlFormValues } from "@/types/order";
import { useFormStyles } from "@/assets/styles/FormStyles";
import axios from "@/helpers/AxiosConfig";
import { APIError } from "@/types/server";
import { orderControlSchema } from "@/helpers/schemas/OrderSchema";
import { SwitchInput } from "@/components/common/fields/SwitchInput";

interface Props {
  orderControl: OrderControl;
}

function OrderControlForm({ orderControl }: Props) {
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
          data.errors.forEach((e: APIError) => {
            form.setFieldError(e.attr, e.detail);
            if (e.attr === "nonFieldErrors") {
              notifications.show({
                title: "Error",
                message: e.detail,
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
    },
  );

  const form = useForm<OrderControlFormValues>({
    validate: yupResolver(orderControlSchema),
    initialValues: {
      autoRateOrders: orderControl.autoRateOrders,
      calculateDistance: orderControl.calculateDistance,
      enforceRevCode: orderControl.enforceRevCode,
      enforceVoidedComm: orderControl.enforceVoidedComm,
      generateRoutes: orderControl.generateRoutes,
      enforceCommodity: orderControl.enforceCommodity,
      autoSequenceStops: orderControl.autoSequenceStops,
      autoOrderTotal: orderControl.autoOrderTotal,
      enforceOriginDestination: orderControl.enforceOriginDestination,
      checkForDuplicateBol: orderControl.checkForDuplicateBol,
      removeOrders: orderControl.removeOrders,
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
            <SwitchInput<OrderControlFormValues>
              form={form}
              className={classes.fields}
              name="autoRateOrders"
              label="Auto Rate Orders"
              description="Automatically rate orders when they are created"
            />
            <SwitchInput<OrderControlFormValues>
              form={form}
              className={classes.fields}
              name="calculateDistance"
              label="Auto Calculate Distance"
              description="Automatically Calculate distance between stops"
            />
            <SwitchInput<OrderControlFormValues>
              form={form}
              className={classes.fields}
              name="enforceRevCode"
              label="Enforce Rev Code"
              description="Enforce rev code code when entering an order."
            />
            <SwitchInput<OrderControlFormValues>
              form={form}
              className={classes.fields}
              name="generateRoutes"
              label="Auto Generate Routes"
              description="Automatically generate routing information for the order."
            />
            <SwitchInput<OrderControlFormValues>
              form={form}
              className={classes.fields}
              name="enforceCommodity"
              label="Enforce Commodity"
              description="Enforce the commodity input on the entry of an order."
            />
            <SwitchInput<OrderControlFormValues>
              form={form}
              className={classes.fields}
              name="autoSequenceStops"
              label="Auto Sequence Stops"
              description="Auto Sequence stops for the order and movements."
            />
            <SwitchInput<OrderControlFormValues>
              form={form}
              className={classes.fields}
              name="autoOrderTotal"
              label="Auto Total Orders"
              description="Automate the order total amount calculation."
            />
            <SwitchInput<OrderControlFormValues>
              form={form}
              className={classes.fields}
              name="enforceOriginDestination"
              label="Enforce Origin Destination"
              description="Compare and validate that origin and destination are not the same."
            />
            <SwitchInput<OrderControlFormValues>
              form={form}
              className={classes.fields}
              name="checkForDuplicateBol"
              label="Check for Duplicate BOL"
              description="Check for duplicate BOL numbers when entering an order."
            />
            <SwitchInput<OrderControlFormValues>
              form={form}
              className={classes.fields}
              name="removeOrders"
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
}

export default function OrderControlPage() {
  const { classes } = usePageStyles();
  const queryClient = useQueryClient();

  const { data: orderControlData, isLoading: isOrderControlDataLoading } =
    useQuery({
      queryKey: ["orderControl"],
      queryFn: () => getOrderControl(),
      initialData: () => queryClient.getQueryData(["orderControl"]),
      staleTime: Infinity,
    });

  // Store first element of orderControlData in variable
  const orderControlDataArray = orderControlData?.[0];

  return isOrderControlDataLoading ? (
    <Skeleton height={400} />
  ) : (
    <Card className={classes.card}>
      <Text fz="xl" fw={700} className={classes.text}>
        Order Controls
      </Text>

      <Divider my={10} />
      {orderControlDataArray && (
        <OrderControlForm orderControl={orderControlDataArray} />
      )}
    </Card>
  );
}
