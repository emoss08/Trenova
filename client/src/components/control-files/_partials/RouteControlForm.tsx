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

import {
  Box,
  Button,
  createStyles,
  Group,
  rem,
  SimpleGrid,
} from "@mantine/core";
import { SelectInput } from "@/components/ui/fields/SelectInput";
import React from "react";
import { useForm, yupResolver } from "@mantine/form";
import * as Yup from "yup";
import { SwitchInput } from "@/components/ui/fields/SwitchInput";
import { useMutation, useQueryClient } from "react-query";
import axios from "@/lib/AxiosConfig";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { APIError } from "@/types/server";
import { ValidatedNumberInput } from "@/components/ui/fields/NumberInput";
import { RouteControl } from "@/types/apps/route";
import {
  distanceMethodChoices,
  routeDistanceUnitChoices,
} from "@/utils/apps/route";

interface RouteControlFormValues {
  distance_method: string;
  mileage_unit: string;
  generate_routes: boolean;
}

interface Props {
  routeControl: RouteControl;
}

const useStyles = createStyles((theme) => {
  const BREAKPOINT = theme.fn.smallerThan("sm");

  return {
    fields: {
      marginTop: rem(20),
    },
    control: {
      [BREAKPOINT]: {
        flex: 1,
      },
    },
    text: {
      color: theme.colorScheme === "dark" ? "white" : "black",
    },
    invalid: {
      backgroundColor:
        theme.colorScheme === "dark"
          ? theme.fn.rgba(theme.colors.red[8], 0.15)
          : theme.colors.red[0],
    },
    invalidIcon: {
      color: theme.colors.red[theme.colorScheme === "dark" ? 7 : 6],
    },
    div: {
      marginBottom: rem(10),
    },
  };
});

export const RouteControlForm: React.FC<Props> = ({ routeControl }) => {
  const { classes } = useStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  console.log(routeControl);
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

  const routeControlSchema = Yup.object().shape({
    distance_method: Yup.string().required("Distance method is required"),
    mileage_unit: Yup.string().required("Mileage unit is required"),
    generate_routes: Yup.boolean().required("Generate routes is required"),
  });

  const form = useForm<RouteControlFormValues>({
    validate: yupResolver(routeControlSchema),
    initialValues: {
      distance_method: routeControl.distance_method,
      mileage_unit: routeControl.mileage_unit,
      generate_routes: routeControl.generate_routes,
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
            <SelectInput
              form={form}
              data={distanceMethodChoices}
              className={classes.fields}
              name="distance_method"
              label="Distance Method"
              placeholder="Distance Method"
              description="Distance method for the company."
              variant="filled"
              withAsterisk
            />
            <SelectInput
              form={form}
              data={routeDistanceUnitChoices}
              className={classes.fields}
              name="mileage_unit"
              label="Mileage Unit"
              placeholder="Mileage Unit"
              description="The mileage unit that the organization uses."
              variant="filled"
              withAsterisk
            />
            <SwitchInput
              form={form}
              className={classes.fields}
              name="generate_routes"
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
};
