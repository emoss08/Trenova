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
import { getDispatchControl } from "@/services/OrganizationRequestService";
import { usePageStyles } from "@/assets/styles/PageStyles";
import { DispatchControl, DispatchControlFormValues } from "@/types/dispatch";
import { useFormStyles } from "@/assets/styles/FormStyles";
import axios from "@/helpers/AxiosConfig";
import { APIError } from "@/types/server";
import { dispatchControlSchema } from "@/helpers/schemas/DispatchSchema";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { ServiceIncidentControlChoices } from "@/helpers/choices";
import { ValidatedNumberInput } from "@/components/common/fields/NumberInput";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { SwitchInput } from "@/components/common/fields/SwitchInput";

interface Props {
  dispatchControl: DispatchControl;
}

function DispatchControlForm({ dispatchControl }: Props): React.ReactElement {
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

  const form = useForm<DispatchControlFormValues>({
    validate: yupResolver(dispatchControlSchema),
    initialValues: {
      recordServiceIncident: dispatchControl.recordServiceIncident,
      gracePeriod: dispatchControl.gracePeriod,
      deadheadTarget: dispatchControl.deadheadTarget,
      driverAssign: dispatchControl.driverAssign,
      trailerContinuity: dispatchControl.trailerContinuity,
      dupeTrailerCheck: dispatchControl.dupeTrailerCheck,
      regulatoryCheck: dispatchControl.regulatoryCheck,
      prevOrdersOnHold: dispatchControl.prevOrdersOnHold,
      driverTimeAwayRestriction: dispatchControl.driverTimeAwayRestriction,
      tractorWorkerFleetConstraint:
        dispatchControl.tractorWorkerFleetConstraint,
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
            <SelectInput<DispatchControlFormValues>
              form={form}
              data={ServiceIncidentControlChoices}
              className={classes.fields}
              name="recordServiceIncident"
              label="Record Service Incident"
              placeholder="Record Service Incident"
              description="Record service incident for the company."
              variant="filled"
              withAsterisk
            />
            <ValidatedNumberInput<DispatchControlFormValues>
              form={form}
              className={classes.fields}
              name="gracePeriod"
              label="Grace Period"
              placeholder="Grace Period"
              description="Grace period for the service incident in minutes."
              variant="filled"
              withAsterisk
            />
            <ValidatedTextInput<DispatchControlFormValues>
              form={form}
              className={classes.fields}
              name="deadheadTarget"
              label="Deadhead Target"
              placeholder="Deadhead Target"
              description="Deadhead target for the company."
              variant="filled"
              withAsterisk
            />
            <SwitchInput<DispatchControlFormValues>
              form={form}
              className={classes.fields}
              name="driverAssign"
              label="Driver Assign"
              description="Enforce driver assign to orders for the company."
            />
            <SwitchInput<DispatchControlFormValues>
              form={form}
              className={classes.fields}
              name="trailerContinuity"
              label="Trailer Continuity"
              description="Enforce trailer continuity for the company."
            />
            <SwitchInput<DispatchControlFormValues>
              form={form}
              className={classes.fields}
              name="dupeTrailerCheck"
              label="Dupe Trailer Check"
              description="Enforce duplicate trailer check for the company."
            />
            <SwitchInput<DispatchControlFormValues>
              form={form}
              className={classes.fields}
              name="regulatoryCheck"
              label="Regulatory Check"
              description="Enforce regulatory check for the company."
            />
            <SwitchInput<DispatchControlFormValues>
              form={form}
              className={classes.fields}
              name="prevOrdersOnHold"
              label="Previous Orders on Hold"
              description="Prevent dispatch of orders on hold for the company."
            />
            <SwitchInput<DispatchControlFormValues>
              form={form}
              className={classes.fields}
              name="driverTimeAwayRestriction"
              label="Driver Time Away Restriction"
              description="Disallow assignments if the driver is on Time Away."
            />
            <SwitchInput<DispatchControlFormValues>
              form={form}
              className={classes.fields}
              name="tractorWorkerFleetConstraint"
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
}

export default function DispatchControlPage(): React.ReactElement {
  const { classes } = usePageStyles();
  const queryClient = useQueryClient();

  const { data: dispatchControlData, isLoading: isDispatchControlDataLoading } =
    useQuery({
      queryKey: ["dispatchControl"],
      queryFn: () => getDispatchControl(),
      initialData: () => queryClient.getQueryData(["dispatchControl"]),
      staleTime: Infinity,
    });

  // Store first element of dispatchControlData in variable
  const dispatchControlDataArray = dispatchControlData?.[0];

  return isDispatchControlDataLoading ? (
    <Skeleton height={400} />
  ) : (
    <Card className={classes.card}>
      <Text fz="xl" fw={700} className={classes.text}>
        Dispatch Controls
      </Text>

      <Divider my={10} />
      {dispatchControlDataArray && (
        <DispatchControlForm dispatchControl={dispatchControlDataArray} />
      )}
    </Card>
  );
}
