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

import { Box, Button, Group, Modal, SimpleGrid, Skeleton } from "@mantine/core";
import React, { Suspense } from "react";
import { notifications } from "@mantine/notifications";
import { useForm, yupResolver } from "@mantine/form";
import { generalLedgerTableStore as store } from "@/stores/AccountingStores";
import { useFormStyles } from "@/assets/styles/FormStyles";
import {
  GeneralLedgerAccount,
  GLAccountFormValues as FormValues,
} from "@/types/accounting";
import { glAccountSchema } from "@/lib/schemas/AccountingSchema";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { statusChoices } from "@/lib/constants";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";
import {
  accountClassificationChoices,
  accountSubTypeChoices,
  accountTypeChoices,
  cashFlowTypeChoices,
} from "@/lib/choices";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";

function CreateGLAccountModalForm(): React.ReactElement {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);

  const form = useForm<FormValues>({
    validate: yupResolver(glAccountSchema),
    initialValues: {
      status: "A",
      accountNumber: "0000-0000-0000-0000", // TODO(WOLFRED): Instead get the next account number from the backend
      description: "",
      accountType: "",
      cashFlowType: "",
      accountSubType: "",
      accountClassification: "",
    },
  });

  const mutation = useCustomMutation<
    FormValues,
    Omit<TableStoreProps<GeneralLedgerAccount>, "drawerOpen">
  >(
    form,
    store,
    notifications,
    {
      method: "POST",
      path: "/gl_accounts/",
      successMessage: "General Ledger Account created successfully.",
      queryKeysToInvalidate: ["gl-account-table-data"],
      additionalInvalidateQueries: ["glAccounts"],
      closeModal: true,
      errorMessage: "Failed to create general ledger account.",
    },
    () => setLoading(false),
  );

  const submitForm = (values: FormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => submitForm(values))}>
      <Box className={classes.div}>
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
          <SelectInput<FormValues>
            form={form}
            data={statusChoices}
            className={classes.fields}
            name="status"
            label="Status"
            placeholder="Status"
            variant="filled"
            withAsterisk
          />
          <ValidatedTextInput<FormValues>
            form={form}
            className={classes.fields}
            name="accountNumber"
            label="Account Number"
            placeholder="Account Number"
            variant="filled"
            withAsterisk
          />
        </SimpleGrid>
        <ValidatedTextArea<FormValues>
          form={form}
          className={classes.fields}
          name="description"
          label="Description"
          placeholder="Description"
          variant="filled"
          withAsterisk
        />
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
          <SelectInput<FormValues>
            form={form}
            data={accountTypeChoices}
            className={classes.fields}
            name="accountType"
            label="Account Type"
            placeholder="AP Account"
            variant="filled"
            withAsterisk
            clearable
          />
          <SelectInput<FormValues>
            form={form}
            data={cashFlowTypeChoices}
            className={classes.fields}
            name="cashFlowType"
            label="Cash Flow Type"
            placeholder="Cash Flow Type"
            variant="filled"
            clearable
          />
        </SimpleGrid>
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
          <SelectInput<FormValues>
            form={form}
            data={accountSubTypeChoices}
            className={classes.fields}
            name="accountSubType"
            label="Account Sub Type"
            placeholder="Account Sub Type"
            variant="filled"
            clearable
          />
          <SelectInput<FormValues>
            form={form}
            data={accountClassificationChoices}
            className={classes.fields}
            name="accountClassification"
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
    </form>
  );
}

export function CreateGLAccountModal(): React.ReactElement {
  const [showCreateModal, setShowCreateModal] = store.use("createModalOpen");

  return (
    <Modal.Root
      opened={showCreateModal}
      onClose={() => setShowCreateModal(false)}
    >
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Create GL Account</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <Suspense fallback={<Skeleton height={400} />}>
            <CreateGLAccountModalForm />
          </Suspense>
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
