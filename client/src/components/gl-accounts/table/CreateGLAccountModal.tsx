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

import { useGLAccounts } from "@/hooks/useGLAccounts";
import { useTags } from "@/hooks/useTags";
import { useUsers } from "@/hooks/useUsers";
import {
  accountClassificationChoices,
  accountSubTypeChoices,
  accountTypeChoices,
  cashFlowTypeChoices,
} from "@/lib/choices";
import { glAccountSchema } from "@/lib/validations/AccountingSchema";
import { generalLedgerTableStore as store } from "@/stores/AccountingStores";
import { TChoiceProps } from "@/types";
import {
  GLAccountFormValues as FormValues,
  GeneralLedgerAccount,
} from "@/types/accounting";
import React, { forwardRef } from "react";

const useStyles = createStyles((theme) => ({
  modalContent: {
    display: "flex",
    flexDirection: "column",
    height: "100%", // This ensures the content stretches the full height of the modal
  },
  modalBody: {
    overflowY: "auto",
    flexGrow: 1, // This ensures the body takes all available space
  },
  stickyButtonGroup: {
    marginTop: theme.spacing.md,
    alignSelf: "flex-end",
    position: "sticky",
    bottom: 0,
    zIndex: 1, // Ensuring the button stays above the content
    backgroundColor: "inherit", // To match the modal's background (can adjust as necessary)
    marginBottom: theme.spacing.md,
    marginRight: theme.spacing.md,
  },
}));

export function GLAccountForm({
  users,
  isUsersLoading,
  isUsersError,
  tags,
  isTagsLoading,
  isTagsError,
  glAccounts,
  isGLAccountsError,
  isGLAccountsLoading,
}: {
  users: TChoiceProps[];
  isUsersLoading: boolean;
  isUsersError: boolean;
  tags: TChoiceProps[];
  isTagsLoading: boolean;
  isTagsError: boolean;
  glAccounts: TChoiceProps[];
  isGLAccountsLoading: boolean;
  isGLAccountsError: boolean;
}) {
  const { classes } = useFormStyles();

  return (
    <div className={classes.div}>
      <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
        <SelectInput<FormValues>
          data={statusChoices}
          name="status"
          label="Status"
          description="Status of the account"
          placeholder="Status"
          variant="filled"
          withAsterisk
        />
        <ValidatedTextInput<FormValues>
          form={form}
          name="accountNumber"
          description="The account number of the account"
          label="Account Number"
          placeholder="Account Number"
          variant="filled"
          withAsterisk
        />
      </SimpleGrid>
      <ValidatedTextArea<FormValues>
        mb={10}
        form={form}
        name="description"
        description="The description of the account"
        label="Description"
        placeholder="Description"
        variant="filled"
        withAsterisk
      />
      <SimpleGrid cols={3} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
        <SelectInput<FormValues>
          form={form}
          data={accountTypeChoices}
          name="accountType"
          label="Account Type"
          description="The type of the account"
          placeholder="AP Account"
          variant="filled"
          withAsterisk
          clearable
        />
        <SelectInput<FormValues>
          form={form}
          data={cashFlowTypeChoices}
          name="cashFlowType"
          description="The cash flow type of the account"
          label="Cash Flow Type"
          placeholder="Cash Flow Type"
          variant="filled"
          clearable
        />
        <SelectInput<FormValues>
          form={form}
          data={accountSubTypeChoices}
          name="accountSubType"
          description="The sub type of the account"
          label="Account Sub Type"
          placeholder="Account Sub Type"
          variant="filled"
          clearable
        />
        <SelectInput<FormValues>
          form={form}
          data={accountClassificationChoices}
          name="accountClassification"
          description="The classification of the account"
          label="Account Classification"
          placeholder="Account Classification"
          variant="filled"
          clearable
        />
        <SelectInput<FormValues>
          form={form}
          data={glAccounts}
          isLoading={isGLAccountsLoading}
          isError={isGLAccountsError}
          name="parentAccount"
          description="Parent account for hierarchical accounting"
          label="Parent Account"
          placeholder="Parent Account"
          variant="filled"
        />
        <ValidatedFileInput<FormValues>
          form={form}
          name="attachment"
          description="Attach relevant documents or receipts"
          label="Attachment"
          placeholder="Attach File"
        />
        <SelectInput<FormValues>
          form={form}
          name="owner"
          data={users}
          isError={isUsersError}
          isLoading={isUsersLoading}
          label="Owner"
          placeholder="Owner"
          description="User responsible for the account"
        />
        <ValidatedTextInput<FormValues>
          form={form}
          name="interestRate"
          description="Interest rate associated with the account"
          label="Interest Rate"
          placeholder="Interest Rate"
          variant="filled"
        />
        <ValidatedMultiSelect<FormValues>
          form={form}
          name="tags"
          data={tags}
          isError={isTagsError}
          isLoading={isTagsLoading}
          description="Tags or labels associated with the account"
          label="Tags"
          placeholder="Tags"
        />
        <SwitchInput<FormValues>
          form={form}
          name="isTaxRelevant"
          label="Is Tax Relevant"
          description="Indicates if the account is relevant for tax calculations"
        />
        <SwitchInput<FormValues>
          form={form}
          name="isReconciled"
          label="Is Reconciled"
          description="Indicates if the account is reconciled"
        />
      </SimpleGrid>
      <ValidatedTextArea<FormValues>
        form={form}
        name="notes"
        description="Additional notes or comments for the account"
        label="Notes"
        placeholder="Notes"
        variant="filled"
      />
    </div>
  );
}

const CreateGLAccountModalForm = forwardRef<
  HTMLFormElement,
  {
    setLoading: (loading: boolean) => void;
    users: TChoiceProps[];
    isUsersLoading: boolean;
    isUsersError: boolean;
    tags: TChoiceProps[];
    isTagsLoading: boolean;
    isTagsError: boolean;
    glAccounts: TChoiceProps[];
    isGLAccountsLoading: boolean;
    isGLAccountsError: boolean;
  }
>(
  (
    {
      setLoading,
      users,
      isUsersLoading,
      isUsersError,
      tags,
      isTagsError,
      isTagsLoading,
      glAccounts,
      isGLAccountsLoading,
      isGLAccountsError,
    },
    ref,
  ) => {
    const form = useForm<FormValues>({
      validate: yupResolver(glAccountSchema),
      initialValues: {
        status: "A",
        accountNumber: "0000-00",
        description: "",
        accountType: "",
        cashFlowType: "",
        accountSubType: "",
        accountClassification: "",
        parentAccount: "",
        isReconciled: false,
        notes: "",
        owner: "",
        isTaxRelevant: false,
        attachment: null,
        interestRate: 0,
        tags: [],
      },
    });

    const mutation = useCustomMutation<
      FormValues,
      TableStoreProps<GeneralLedgerAccount>
    >(
      form,
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
      store,
    );

    const submitForm = (values: FormValues) => {
      setLoading(true);
      mutation.mutate(values);
    };

    return (
      <form ref={ref} onSubmit={form.onSubmit((values) => submitForm(values))}>
        <GLAccountForm
          form={form}
          users={users}
          isUsersError={isUsersError}
          isUsersLoading={isUsersLoading}
          tags={tags}
          isTagsError={isTagsError}
          isTagsLoading={isTagsLoading}
          glAccounts={glAccounts}
          isGLAccountsLoading={isGLAccountsLoading}
          isGLAccountsError={isGLAccountsError}
        />
      </form>
    );
  },
);

CreateGLAccountModalForm.displayName = "CreateGLAccountModalForm";

export function CreateGLAccountModal(): React.ReactElement {
  const [showCreateModal, setShowCreateModal] = store.use("createModalOpen");
  const { classes } = useStyles();
  const { classes: formStyles } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const formRef = React.useRef<HTMLFormElement>(null);

  const {
    selectUsersData,
    isError: usersError,
    isLoading: usersLoading,
  } = useUsers(showCreateModal);

  const {
    selectTags,
    isError: tagsError,
    isLoading: tagsLoading,
  } = useTags(showCreateModal);

  const {
    selectGLAccounts,
    isError: glAccountsError,
    isLoading: glAccountsLoading,
  } = useGLAccounts(showCreateModal);

  const handleButtonClick = () => {
    if (formRef.current) {
      formRef.current.requestSubmit();
    }
  };
  return (
    <Modal.Root
      opened={showCreateModal}
      onClose={() => setShowCreateModal(false)}
      size="50%"
    >
      <Modal.Overlay />
      <Modal.Content className={classes.modalContent}>
        <Modal.Header>
          <Modal.Title>Create GL Account</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body className={classes.modalBody}>
          <CreateGLAccountModalForm
            ref={formRef}
            setLoading={setLoading}
            users={selectUsersData}
            isUsersLoading={usersLoading}
            isUsersError={usersError}
            tags={selectTags}
            isTagsLoading={tagsLoading}
            isTagsError={tagsError}
            glAccounts={selectGLAccounts}
            isGLAccountsLoading={glAccountsLoading}
            isGLAccountsError={glAccountsError}
          />
        </Modal.Body>
        <Group position="right" mt="md" className={classes.stickyButtonGroup}>
          <Button
            color="white"
            type="submit"
            onClick={handleButtonClick}
            className={formStyles.control}
            loading={loading}
          >
            Submit
          </Button>
        </Group>
      </Modal.Content>
    </Modal.Root>
  );
}
