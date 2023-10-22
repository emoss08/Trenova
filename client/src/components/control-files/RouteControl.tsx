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

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
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
import { getRouteControl } from "@/services/OrganizationRequestService";
import { usePageStyles } from "@/assets/styles/PageStyles";
import { RouteControl, RouteControlFormValues } from "@/types/route";
import { useFormStyles } from "@/assets/styles/FormStyles";
import axios from "@/lib/axiosConfig";
import { APIError } from "@/types/server";
import { routeControlSchema } from "@/lib/validations/RouteSchema";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { SwitchInput } from "@/components/common/fields/SwitchInput";
import { distanceMethodChoices, routeDistanceUnitChoices } from "@/lib/choices";

interface Props {
  routeControl: RouteControl;
}

function RouteControlForm({ routeControl }: Props) {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: RouteControlFormValues) =>
      axios.put(`/route_control/${routeControl.id}/`, values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["routeControl"],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "Route Control updated successfully",
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

  const form = useForm<RouteControlFormValues>({
    validate: yupResolver(routeControlSchema),
    initialValues: {
      distanceMethod: routeControl.distanceMethod,
      mileageUnit: routeControl.mileageUnit,
      generateRoutes: routeControl.generateRoutes,
    },
  });

  const handleSubmit = (values: RouteControlFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => handleSubmit(values))}>
      <Box className={classes.div}>
        <Box>
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <SelectInput<RouteControlFormValues>
              form={form}
              data={distanceMethodChoices}
              className={classes.fields}
              name="distanceMethod"
              label="Distance Method"
              placeholder="Distance Method"
              description="Distance method for the company."
              variant="filled"
              withAsterisk
            />
            <SelectInput<RouteControlFormValues>
              form={form}
              data={routeDistanceUnitChoices}
              className={classes.fields}
              name="mileageUnit"
              label="Mileage Unit"
              placeholder="Mileage Unit"
              description="The mileage unit that the organization uses."
              variant="filled"
              withAsterisk
            />
            <SwitchInput<RouteControlFormValues>
              form={form}
              className={classes.fields}
              name="generateRoutes"
              label="Auto Generate Routes"
              description="Automatically generate routes for the company."
              variant="filled"
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

export default function RouteControlPage() {
  const { classes } = usePageStyles();
  const queryClient = useQueryClient();

  const { data: routeControlData, isLoading: isRouteControlDataLoading } =
    useQuery({
      queryKey: ["routeControl"],
      queryFn: () => getRouteControl(),
      initialData: () => queryClient.getQueryData(["routeControl"]),
      staleTime: Infinity,
    });

  // Store first element of dispatchControlData in variable
  const routeControlDataArray = routeControlData?.[0];

  return isRouteControlDataLoading ? (
    <Skeleton height={400} />
  ) : (
    <Card className={classes.card}>
      <Text fz="xl" fw={700} className={classes.text}>
        Route Controls
      </Text>

      <Divider my={10} />
      {routeControlDataArray && (
        <RouteControlForm routeControl={routeControlDataArray} />
      )}
    </Card>
  );
}
