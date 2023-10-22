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
import { getBillingControl } from "@/services/OrganizationRequestService";
import { usePageStyles } from "@/assets/styles/PageStyles";
import { BillingControl, BillingControlFormValues } from "@/types/billing";
import { useFormStyles } from "@/assets/styles/FormStyles";
import axios from "@/lib/axiosConfig";
import { APIError } from "@/types/server";
import { billingControlSchema } from "@/lib/validations/BillingSchema";
import { SwitchInput } from "@/components/common/fields/SwitchInput";
import { SelectInput } from "@/components/common/fields/SelectInput";
import {
  AutoBillingCriteriaChoices,
  OrderTransferCriteriaChoices,
} from "@/utils/apps/billing";

interface Props {
  billingControl: BillingControl;
}

function BillingControlForm({ billingControl }: Props): React.ReactElement {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: BillingControlFormValues) =>
      axios.put(`/billing_control/${billingControl.id}/`, values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["billingControl"],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "Billing Control updated successfully",
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

  const form = useForm<BillingControlFormValues>({
    validate: yupResolver(billingControlSchema),
    initialValues: {
      removeBillingHistory: billingControl.removeBillingHistory,
      autoBillOrders: billingControl.autoBillOrders,
      autoMarkReadyToBill: billingControl.autoMarkReadyToBill,
      validateCustomerRates: billingControl.validateCustomerRates,
      autoBillCriteria: billingControl.autoBillCriteria,
      orderTransferCriteria: billingControl.orderTransferCriteria,
      enforceCustomerBilling: billingControl.enforceCustomerBilling,
    },
  });

  const handleSubmit = (values: BillingControlFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => handleSubmit(values))}>
      <Box className={classes.div}>
        <Box>
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <SwitchInput<BillingControlFormValues>
              form={form}
              className={classes.fields}
              name="removeBillingHistory"
              label="Remove Billing History"
              description="Whether users can remove records from billing history."
            />
            <SwitchInput<BillingControlFormValues>
              form={form}
              className={classes.fields}
              name="autoBillOrders"
              label="Auto Bill Orders"
              description="Whether to automatically bill orders directly to customer."
            />
            <SwitchInput<BillingControlFormValues>
              form={form}
              className={classes.fields}
              name="autoMarkReadyToBill"
              label="Auto Mark Ready to Bill"
              description="Marks orders as ready to bill when they are delivered and meet customer billing requirements."
            />
            <SwitchInput<BillingControlFormValues>
              form={form}
              className={classes.fields}
              name="validateCustomerRates"
              label="Validate Customer Rates"
              description="Validate rates match the customer contract in the billing queue before allowing billing."
            />
            <SelectInput<BillingControlFormValues>
              form={form}
              data={AutoBillingCriteriaChoices}
              className={classes.fields}
              name="autoBillCriteria"
              label="Auto Bill Criteria"
              placeholder="Auto Bill Criteria"
              description="Define a criteria on when auto billing is to occur."
              variant="filled"
              clearable
            />
            <SelectInput<BillingControlFormValues>
              form={form}
              data={OrderTransferCriteriaChoices}
              className={classes.fields}
              name="orderTransferCriteria"
              label="Order Transfer Criteria"
              placeholder="Order Transfer Criteria"
              description="Define a criteria on when orders are to be transferred."
              variant="filled"
            />
            <SwitchInput<BillingControlFormValues>
              form={form}
              className={classes.fields}
              name="enforceCustomerBilling"
              label="Enforce Customer Billing"
              description="Define if customer billing requirements will be enforced when billing."
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

export default function BillingControlPage(): React.ReactElement {
  const { classes } = usePageStyles();
  const queryClient = useQueryClient();

  const { data: billingControlData, isLoading: isBillingControlDataLoading } =
    useQuery({
      queryKey: ["billingControl"],
      queryFn: () => getBillingControl(),
      initialData: () => queryClient.getQueryData(["billingControl"]),
      staleTime: Infinity,
    });

  // Store first element of BillingControlData in variable
  const billingControlDataArray = billingControlData?.[0];

  return isBillingControlDataLoading ? (
    <Skeleton height={400} />
  ) : (
    <Card className={classes.card}>
      <Text fz="xl" fw={700} className={classes.text}>
        Billing Controls
      </Text>

      <Divider my={10} />
      {billingControlDataArray && (
        <BillingControlForm billingControl={billingControlDataArray} />
      )}
    </Card>
  );
}
