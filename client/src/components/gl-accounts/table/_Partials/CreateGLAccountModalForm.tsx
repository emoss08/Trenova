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
import { Box, Button, Group, SimpleGrid } from "@mantine/core";
import { useMutation, useQueryClient } from "react-query";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useForm, yupResolver } from "@mantine/form";
import { SelectInput } from "@/components/ui/fields/SelectInput";
import { statusChoices } from "@/lib/utils";
import { ValidatedTextInput } from "@/components/ui/fields/TextInput";
import { ValidatedTextArea } from "@/components/ui/fields/TextArea";
import axios from "@/lib/AxiosConfig";
import { APIError } from "@/types/server";
import { generalLedgerTableStore } from "@/stores/AccountingStores";
import {
  accountClassificationChoices,
  accountSubTypeChoices,
  accountTypeChoices,
  cashFlowTypeChoices,
} from "@/utils/apps/accounting";
import { useFormStyles } from "@/styles/FormStyles";
import { GLAccountFormValues } from "@/types/apps/accounting";
import { glAccountSchema } from "@/utils/apps/accounting/schema";

export const CreateGLAccountModalForm: React.FC = () => {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: GLAccountFormValues) => axios.post("/gl_accounts/", values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["gl-account-table-data"],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "GL Account created successfully",
              color: "green",
              withCloseButton: true,
              icon: <FontAwesomeIcon icon={faCheck} />,
            });
            generalLedgerTableStore.set("createModalOpen", false);
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

  const form = useForm<GLAccountFormValues>({
    validate: yupResolver(glAccountSchema),
    initialValues: {
      status: "A",
      account_number: "0000-0000-0000-0000",
      description: "",
      account_type: "",
      cash_flow_type: "",
      account_sub_type: "",
      account_classification: "",
    },
  });

  const submitForm = (values: GLAccountFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => submitForm(values))}>
      <Box className={classes.div}>
        <Box>
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <SelectInput
              form={form}
              data={statusChoices}
              className={classes.fields}
              name="status"
              label="Status"
              placeholder="Status"
              variant="filled"
              withAsterisk
            />
            <ValidatedTextInput
              form={form}
              className={classes.fields}
              name="account_number"
              label="Account Number"
              placeholder="Account Number"
              variant="filled"
              withAsterisk
            />
          </SimpleGrid>
          <ValidatedTextArea
            form={form}
            className={classes.fields}
            name="description"
            label="Description"
            placeholder="Description"
            variant="filled"
            withAsterisk
          />
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <SelectInput
              form={form}
              data={accountTypeChoices}
              className={classes.fields}
              name="account_type"
              label="Account Type"
              placeholder="AP Account"
              variant="filled"
              withAsterisk
              clearable
            />
            <SelectInput
              form={form}
              data={cashFlowTypeChoices}
              className={classes.fields}
              name="cash_flow_type"
              label="Cash Flow Type"
              placeholder="Cash Flow Type"
              variant="filled"
              clearable
            />
          </SimpleGrid>
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <SelectInput
              form={form}
              data={accountSubTypeChoices}
              className={classes.fields}
              name="account_sub_type"
              label="Account Sub Type"
              placeholder="Account Sub Type"
              variant="filled"
              clearable
            />
            <SelectInput
              form={form}
              data={accountClassificationChoices}
              className={classes.fields}
              name="account_classification"
              label="Account Classification"
              placeholder="Account Classification"
              variant="filled"
              clearable
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
