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

import { Box, Button, Group, Modal, SimpleGrid } from "@mantine/core";
import React from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { notifications } from "@mantine/notifications";
import { useForm, yupResolver } from "@mantine/form";
import { revenueCodeTableStore as store } from "@/stores/AccountingStores";
import { getGLAccounts } from "@/services/AccountingRequestService";
import {
  GeneralLedgerAccount,
  RevenueCode,
  RevenueCodeFormValues,
} from "@/types/accounting";
import { TChoiceProps } from "@/types";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { revenueCodeSchema } from "@/lib/validations/AccountingSchema";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";

type CreateRCModalFormProps = {
  selectGlAccountData: TChoiceProps[];
};

function CreateRCModalForm({
  selectGlAccountData,
}: CreateRCModalFormProps): React.ReactElement {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);

  const form = useForm<RevenueCodeFormValues>({
    validate: yupResolver(revenueCodeSchema),
    initialValues: {
      code: "",
      description: "",
      expenseAccount: "",
      revenueAccount: "",
    },
  });

  const mutation = useCustomMutation<
    RevenueCodeFormValues,
    TableStoreProps<RevenueCode>
  >(
    form,
    notifications,
    {
      method: "POST",
      path: "/revenue_codes/",
      successMessage: "Revenue Code created successfully.",
      queryKeysToInvalidate: ["revenue-code-table-data"],
      additionalInvalidateQueries: ["revenueCodes"],
      closeModal: true,
      errorMessage: "Failed to create revenue code.",
    },
    () => setLoading(false),
    store,
  );

  const submitForm = (values: RevenueCodeFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => submitForm(values))}>
      <Box className={classes.div}>
        <ValidatedTextInput<RevenueCodeFormValues>
          form={form}
          className={classes.fields}
          name="code"
          label="Code"
          placeholder="Code"
          variant="filled"
          withAsterisk
        />
        <ValidatedTextArea<RevenueCodeFormValues>
          form={form}
          className={classes.fields}
          name="description"
          label="Description"
          placeholder="Description"
          variant="filled"
          withAsterisk
        />
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
          <SelectInput<RevenueCodeFormValues>
            form={form}
            data={selectGlAccountData}
            className={classes.fields}
            name="expenseAccount"
            label="Expense Account"
            placeholder="Expense Account"
            variant="filled"
            clearable
          />
          <SelectInput<RevenueCodeFormValues>
            form={form}
            data={selectGlAccountData}
            className={classes.fields}
            name="revenueAccount"
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
    </form>
  );
}

export function CreateRCModal(): React.ReactElement {
  const [showCreateModal, setShowCreateModal] = store.use("createModalOpen");
  const queryClient = useQueryClient();

  const { data: glAccountData } = useQuery({
    queryKey: "gl-account-data",
    queryFn: () => getGLAccounts(),
    enabled: showCreateModal,
    initialData: () => queryClient.getQueryData("gl-account-data"),
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
          <Modal.Title>Create Revenue Code</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <CreateRCModalForm selectGlAccountData={selectGlAccountData} />
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
