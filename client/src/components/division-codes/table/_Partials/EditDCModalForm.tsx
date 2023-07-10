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

import { DivisionCode, DivisionCodeFormValues } from "@/types/apps/accounting";
import React from "react";
import { Box, Button, Group, SimpleGrid } from "@mantine/core";
import { SelectInput } from "@/components/ui/fields/SelectInput";
import { statusChoices } from "@/lib/utils";
import { ValidatedTextInput } from "@/components/ui/fields/TextInput";
import { ValidatedTextArea } from "@/components/ui/fields/TextArea";
import { useMutation, useQueryClient } from "react-query";
import axios from "@/lib/AxiosConfig";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { APIError } from "@/types/server";
import { useForm, yupResolver } from "@mantine/form";
import { divisionCodeTableStore } from "@/stores/AccountingStores";
import { TChoiceProps } from "@/types";
import { divisionCodeSchema } from "@/utils/apps/accounting/schema";
import { useFormStyles } from "@/styles/FormStyles";

type Props = {
  divisionCode: DivisionCode;
  selectGlAccountData: TChoiceProps[];
};

export const EditDCModalForm: React.FC<Props> = ({
  divisionCode,
  selectGlAccountData,
}) => {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: DivisionCodeFormValues) =>
      axios.put(`/division_codes/${divisionCode.id}/`, values),
    {
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: ["division-code-table-data"],
        });
        queryClient
          .invalidateQueries({
            queryKey: ["divisionCode", divisionCode?.id],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "Division Code updated successfully",
              color: "green",
              withCloseButton: true,
              icon: <FontAwesomeIcon icon={faCheck} />,
            });
            divisionCodeTableStore.set("editModalOpen", false);
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

  const form = useForm<DivisionCodeFormValues>({
    validate: yupResolver(divisionCodeSchema),
    initialValues: {
      status: divisionCode.status,
      code: divisionCode.code,
      description: divisionCode.description,
      ap_account: divisionCode.ap_account || "",
      cash_account: divisionCode.cash_account || "",
      expense_account: divisionCode.expense_account || "",
    },
  });

  const submitForm = (values: DivisionCodeFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <>
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
                onMouseLeave={() => {
                  form.setFieldValue("status", form.values.status);
                }}
              />
              <ValidatedTextInput
                form={form}
                className={classes.fields}
                name="code"
                label="Code"
                placeholder="Code"
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
                data={selectGlAccountData}
                className={classes.fields}
                name="ap_account"
                label="AP Account"
                placeholder="AP Account"
                variant="filled"
                onMouseLeave={() => {
                  form.setFieldValue("ap_account", form.values.ap_account);
                }}
                clearable
              />
              <SelectInput
                form={form}
                data={selectGlAccountData}
                className={classes.fields}
                name="cash_account"
                label="Cash Account"
                placeholder="Cash Account"
                variant="filled"
                onMouseLeave={() => {
                  form.setFieldValue("cash_account", form.values.cash_account);
                }}
                clearable
              />
            </SimpleGrid>
            <SelectInput
              form={form}
              data={selectGlAccountData}
              className={classes.fields}
              name="expense_account"
              label="Expense Account"
              placeholder="Expense Account"
              variant="filled"
              clearable
            />
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
    </>
  );
};
