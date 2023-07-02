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

import { GeneralLedgerAccount } from "@/types/apps/accounting";
import React from "react";
import {
  Box,
  Button,
  createStyles,
  Group,
  rem,
  SimpleGrid,
} from "@mantine/core";
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
import * as Yup from "yup";
import { useForm, yupResolver } from "@mantine/form";
import { generalLedgerTableStore } from "@/stores/AccountingStores";
import {
  accountClassificationChoices,
  accountSubTypeChoices,
  accountTypeChoices,
  cashFlowTypeChoices,
} from "@/utils/apps/accounting";

type Props = {
  glAccount: GeneralLedgerAccount;
};

interface EditGLAccountFormValues {
  status: string;
  account_number: string;
  description: string;
  account_type: string;
  cash_flow_type?: string;
  account_sub_type?: string;
  account_classification?: string;
}

const useStyles = createStyles((theme) => {
  const BREAKPOINT = theme.fn.smallerThan("sm");

  return {
    fields: {
      marginTop: rem(10),
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

export const EditGLAccountModalForm: React.FC<Props> = ({ glAccount }) => {
  const { classes } = useStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: EditGLAccountFormValues) =>
      axios.put(`/gl_accounts/${glAccount.id}/`, values),
    {
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: ["gl-account-table-data"],
        });
        queryClient
          .invalidateQueries({
            queryKey: ["glAccount", glAccount.id],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "GL Account updated successfully",
              color: "green",
              withCloseButton: true,
              icon: <FontAwesomeIcon icon={faCheck} />,
            });
            generalLedgerTableStore.set("editModalOpen", false);
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

  const editGLAccountSchema = Yup.object().shape({
    status: Yup.string().required("Status is required"),
    account_number: Yup.string()
      .required("Code is required")
      .test(
        "account_number_format",
        "Account number must be in the format 0000-0000-0000-0000",
        (value) => {
          if (!value) return false;
          const regex = /^[0-9]{4}-[0-9]{4}-[0-9]{4}-[0-9]{4}$/;
          return regex.test(value);
        }
      ),
    description: Yup.string()
      .max(100, "Description cannot be longer than 100 characters")
      .required("Description is required"),
    account_type: Yup.string().required("Account type is required"),
    cash_flow_type: Yup.string().notRequired(),
    account_sub_type: Yup.string().notRequired(),
    account_classification: Yup.string().notRequired(),
  });

  const form = useForm<EditGLAccountFormValues>({
    validate: yupResolver(editGLAccountSchema),
    initialValues: {
      status: glAccount.status,
      account_number: glAccount.account_number,
      description: glAccount.description,
      account_type: glAccount.account_type,
      cash_flow_type: glAccount?.cash_flow_type || "",
      account_sub_type: glAccount?.account_sub_type || "",
      account_classification: glAccount?.account_classification || "",
    },
  });

  const submitForm = (values: EditGLAccountFormValues) => {
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
                onMouseLeave={() => {
                  form.setFieldValue("account_type", form.values.account_type);
                }}
                clearable
                withAsterisk
              />
              <SelectInput
                form={form}
                data={cashFlowTypeChoices}
                className={classes.fields}
                name="cash_flow_type"
                label="Cash Flow Type"
                placeholder="Cash Flow Type"
                variant="filled"
                onMouseLeave={() => {
                  form.setFieldValue(
                    "cash_flow_type",
                    form.values.cash_flow_type
                  );
                }}
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
                onMouseLeave={() => {
                  form.setFieldValue(
                    "account_sub_type",
                    form.values.account_sub_type
                  );
                }}
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
                onMouseLeave={() => {
                  form.setFieldValue(
                    "account_classification",
                    form.values.account_classification
                  );
                }}
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
    </>
  );
};