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
import { ValidatedTextInput } from "@/components/ui/fields/TextInput";
import { ValidatedTextArea } from "@/components/ui/fields/TextArea";
import { SelectInput } from "@/components/ui/fields/SelectInput";
import React from "react";
import { useMutation, useQueryClient } from "react-query";
import axios from "@/lib/AxiosConfig";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useForm, yupResolver } from "@mantine/form";
import { revenueCodeTableStore } from "@/stores/AccountingStores";
import { ChoiceProps } from "@/types";
import { RevenueCodeFormValues } from "@/types/apps/accounting";
import { revenueCodeSchema } from "@/utils/apps/accounting/schema";

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

type Props = {
  selectGlAccountData: ChoiceProps[];
};

export const CreateRCModalForm: React.FC<Props> = ({ selectGlAccountData }) => {
  const { classes } = useStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: RevenueCodeFormValues) => axios.post(`/revenue_codes/`, values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["revenue-code-table-data"],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "Revenue Code created successfully",
              color: "green",
              withCloseButton: true,
              icon: <FontAwesomeIcon icon={faCheck} />,
            });
            revenueCodeTableStore.set("createModalOpen", false);
          });
      },
      onError: (error: any) => {
        const { data } = error.response;
        console.log(data);
        if (data.type === "validation_error") {
          data.errors.forEach((error: any) => {
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

  const form = useForm<RevenueCodeFormValues>({
    validate: yupResolver(revenueCodeSchema),
    initialValues: {
      code: "",
      description: "",
      expense_account: "",
      revenue_account: "",
    },
  });

  const submitForm = (values: RevenueCodeFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <>
      <form onSubmit={form.onSubmit((values) => submitForm(values))}>
        <Box className={classes.div}>
          <Box>
            <ValidatedTextInput
              form={form}
              className={classes.fields}
              name="code"
              label="Code"
              placeholder="Code"
              variant="filled"
              withAsterisk
            />
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
                name="expense_account"
                label="Expense Account"
                placeholder="Expense Account"
                variant="filled"
                clearable
              />
              <SelectInput
                form={form}
                data={selectGlAccountData}
                className={classes.fields}
                name="revenue_account"
                label="Revenue Account"
                placeholder="Revenue Account"
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
    </>
  );
};
