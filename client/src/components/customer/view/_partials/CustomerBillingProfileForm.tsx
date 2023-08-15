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
import { useMutation, useQuery, useQueryClient } from "react-query";
import { useForm, yupResolver } from "@mantine/form";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { Box, Button, Divider, Group, SimpleGrid, Text } from "@mantine/core";
import { useFormStyles } from "@/styles/FormStyles";
import axios from "@/lib/AxiosConfig";
import { APIError } from "@/types/server";
import {
  CustomerBillingProfile,
  CustomerBillingProfileFormValues,
  CustomerEmailProfile,
  CustomerRuleProfile,
} from "@/types/apps/customer";
import { SelectInput } from "@/components/ui/fields/SelectInput";
import { statusChoices } from "@/lib/utils";
import { customerStore as store } from "@/stores/CustomerStore";
import {
  getCustomerEmailProfiles,
  getCustomerRuleProfiles,
} from "@/requests/CustomerRequestFactory";
import { customerBillingProfileSchema } from "@/utils/apps/customers/schema";

type Props = {
  billingProfile: CustomerBillingProfile;
};

export function CustomerBillingProfileForm({ billingProfile }: Props) {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const { data: customerRuleProfiles } = useQuery({
    queryKey: "customerRuleProfiles",
    queryFn: () => getCustomerRuleProfiles(),
    enabled: store.get("activeTab") === "profile",
    initialData: () => queryClient.getQueryData("customerRuleProfiles"),
    staleTime: Infinity,
  });

  const ruleProfiles =
    customerRuleProfiles?.map((customerRuleProfile: CustomerRuleProfile) => ({
      value: customerRuleProfile.id,
      label: customerRuleProfile.name,
    })) || [];

  const { data: customerEmailProfiles } = useQuery({
    queryKey: "customerEmailProfiles",
    queryFn: () => getCustomerEmailProfiles(),
    enabled: store.get("activeTab") === "profile",
    initialData: () => queryClient.getQueryData("customerEmailProfiles"),
    staleTime: Infinity,
  });

  const emailProfiles =
    customerEmailProfiles?.map(
      (customerEmailProfile: CustomerEmailProfile) => ({
        value: customerEmailProfile.id,
        label: customerEmailProfile.name,
      }),
    ) || [];

  const mutation = useMutation(
    (values: CustomerBillingProfileFormValues) =>
      axios.put(`/customer_billing_profiles/${billingProfile.id}/`, values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["customerBillingProfile", billingProfile.customer],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "Customer Billing Profile updated successfully",
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
            if (e.attr === "non_field_errors") {
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

  const form = useForm<CustomerBillingProfileFormValues>({
    validate: yupResolver(customerBillingProfileSchema),
    initialValues: {
      status: billingProfile.status,
      email_profile: billingProfile.email_profile,
      rule_profile: billingProfile.rule_profile,
    },
  });

  const submitForm = (values: CustomerBillingProfileFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <Box>
      <Text className={classes.text} fw={600} fz={20}>
        Customer Billing Profile
      </Text>
      <form onSubmit={form.onSubmit((values) => submitForm(values))}>
        <Box className={classes.div}>
          <Divider my={10} />
          <SimpleGrid cols={3} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <SelectInput
              form={form}
              className={classes.fields}
              name="status"
              placeholder="Status"
              data={statusChoices}
              variant="filled"
              label="Status"
              withAsterisk
            />
            <SelectInput
              className={classes.fields}
              form={form}
              name="email_profile"
              placeholder="Email Profile"
              data={emailProfiles || []}
              variant="filled"
              label="Email Profile"
            />
            <SelectInput
              className={classes.fields}
              form={form}
              name="rule_profile"
              placeholder="Rule Profile"
              data={ruleProfiles || []}
              variant="filled"
              label="Rule Profile"
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
      </form>
    </Box>
  );
}
