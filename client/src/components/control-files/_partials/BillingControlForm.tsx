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
import { BillingControl, BillingControlFormValues } from "@/types/apps/billing";
import { useForm, yupResolver } from "@mantine/form";
import { SwitchInput } from "@/components/ui/fields/SwitchInput";
import {
  autoBillingCriteriaChoices,
  orderTransferCriteriaChoices,
} from "@/utils/apps/billing";
import { useMutation, useQueryClient } from "react-query";
import axios from "@/lib/AxiosConfig";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { APIError } from "@/types/server";
import { billingControlSchema } from "@/utils/apps/billing/schema";

interface Props {
  billingControl: BillingControl;
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

export const BillingControlForm: React.FC<Props> = ({ billingControl }) => {
  const { classes } = useStyles();
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

  const form = useForm<BillingControlFormValues>({
    validate: yupResolver(billingControlSchema),
    initialValues: {
      remove_billing_history: billingControl.remove_billing_history,
      auto_bill_orders: billingControl.auto_bill_orders,
      auto_mark_ready_to_bill: billingControl.auto_mark_ready_to_bill,
      validate_customer_rates: billingControl.validate_customer_rates,
      auto_bill_criteria: billingControl.auto_bill_criteria,
      order_transfer_criteria: billingControl.order_transfer_criteria,
      enforce_customer_billing: billingControl.enforce_customer_billing,
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
            <SwitchInput
              form={form}
              className={classes.fields}
              name="remove_billing_history"
              label="Remove Billing History"
              description="Whether users can remove records from billing history."
            />
            <SwitchInput
              form={form}
              className={classes.fields}
              name="auto_bill_orders"
              label="Auto Bill Orders"
              description="Whether to automatically bill orders directly to customer."
            />
            <SwitchInput
              form={form}
              className={classes.fields}
              name="auto_mark_ready_to_bill"
              label="Auto Mark Ready to Bill"
              description="Marks orders as ready to bill when they are delivered and meet customer billing requirements."
            />
            <SwitchInput
              form={form}
              className={classes.fields}
              name="validate_customer_rates"
              label="Validate Customer Rates"
              description="Validate rates match the customer contract in the billing queue before allowing billing."
            />
            <SelectInput
              form={form}
              data={autoBillingCriteriaChoices}
              className={classes.fields}
              name="auto_bill_criteria"
              label="Auto Bill Criteria"
              placeholder="Auto Bill Criteria"
              description="Define a criteria on when auto billing is to occur."
              variant="filled"
              clearable
            />
            <SelectInput
              form={form}
              data={orderTransferCriteriaChoices}
              className={classes.fields}
              name="order_transfer_criteria"
              label="Order Transfer Criteria"
              placeholder="Order Transfer Criteria"
              description="Define a criteria on when orders are to be transferred."
              variant="filled"
            />
            <SwitchInput
              form={form}
              className={classes.fields}
              name="enforce_customer_billing"
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
};
