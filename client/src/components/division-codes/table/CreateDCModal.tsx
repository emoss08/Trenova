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
  Group,
  Modal,
  SimpleGrid,
  Skeleton,
  Stack,
} from "@mantine/core";
import React from "react";
import { useMutation, useQuery, useQueryClient } from "react-query";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useForm, yupResolver } from "@mantine/form";
import { divisionCodeTableStore as store } from "@/stores/AccountingStores";
import { getGLAccounts } from "@/services/AccountingRequestService";
import {
  DivisionCodeFormValues,
  GeneralLedgerAccount,
} from "@/types/accounting";
import { TChoiceProps } from "@/types";
import { useFormStyles } from "@/assets/styles/FormStyles";
import axios from "@/helpers/AxiosConfig";
import { divisionCodeSchema } from "@/helpers/schemas/AccountingSchema";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { statusChoices } from "@/helpers/constants";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";
import { APIError } from "@/types/server";

type CreateDCModalFormProps = {
  selectGlAccountData: TChoiceProps[];
};

function CreateDCModalForm({ selectGlAccountData }: CreateDCModalFormProps) {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: DivisionCodeFormValues) => axios.post("/division_codes/", values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["division-code-table-data"],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "Division Code created successfully",
              color: "green",
              withCloseButton: true,
              icon: <FontAwesomeIcon icon={faCheck} />,
            });
            store.set("createModalOpen", false);
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
            } else if (
              e.attr === "__all__" &&
              e.detail ===
                "Division Code with this Code and Organization already exists."
            ) {
              form.setFieldError("code", e.detail);
            }
          });
        }
      },
      onSettled: () => {
        setLoading(false);
      },
    },
  );

  const form = useForm<DivisionCodeFormValues>({
    validate: yupResolver(divisionCodeSchema),
    initialValues: {
      status: "A",
      code: "",
      description: "",
      apAccount: "",
      cashAccount: "",
      expenseAccount: "",
    },
  });

  const submitForm = (values: DivisionCodeFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => submitForm(values))}>
      <Box className={classes.div}>
        <Box>
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <SelectInput<DivisionCodeFormValues>
              form={form}
              data={statusChoices}
              name="status"
              label="Status"
              placeholder="Status"
              variant="filled"
              withAsterisk
            />
            <ValidatedTextInput<DivisionCodeFormValues>
              form={form}
              name="code"
              label="Code"
              placeholder="Code"
              variant="filled"
              withAsterisk
              maxLength={4}
            />
          </SimpleGrid>
          <ValidatedTextArea<DivisionCodeFormValues>
            form={form}
            name="description"
            label="Description"
            placeholder="Description"
            variant="filled"
            withAsterisk
          />
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <SelectInput<DivisionCodeFormValues>
              form={form}
              data={selectGlAccountData}
              name="apAccount"
              label="AP Account"
              placeholder="AP Account"
              variant="filled"
              clearable
            />
            <SelectInput<DivisionCodeFormValues>
              form={form}
              data={selectGlAccountData}
              name="cashAccount"
              label="Cash Account"
              placeholder="Cash Account"
              variant="filled"
              clearable
            />
          </SimpleGrid>
          <SelectInput<DivisionCodeFormValues>
            form={form}
            data={selectGlAccountData}
            name="expenseAccount"
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
  );
}

export function CreateDCModal(): React.ReactElement {
  const [showCreateModal, setShowCreateModal] = store.use("createModalOpen");
  const queryClient = useQueryClient();

  const { data: glAccountData, isLoading: isGLAccountDataLoading } = useQuery({
    queryKey: "gl-account-data",
    queryFn: () => getGLAccounts(),
    enabled: showCreateModal,
    initialData: () => queryClient.getQueryData("gl-account"),
    staleTime: Infinity,
  });

  const selectGlAccountData =
    glAccountData?.map((glAccount: GeneralLedgerAccount) => ({
      value: glAccount.id,
      label: glAccount.accountNumber,
    })) || [];

  return (
    <Modal.Root
      opened={showCreateModal}
      onClose={() => setShowCreateModal(false)}
    >
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Create Division Code</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          {isGLAccountDataLoading ? (
            <Stack>
              <Skeleton height={400} />
            </Stack>
          ) : (
            <CreateDCModalForm selectGlAccountData={selectGlAccountData} />
          )}
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
