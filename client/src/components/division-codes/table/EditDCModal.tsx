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
import { useMutation, useQuery, useQueryClient } from "react-query";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useForm, yupResolver } from "@mantine/form";
import { divisionCodeTableStore } from "@/stores/AccountingStores";
import { getGLAccounts } from "@/requests/AccountingRequestFactory";
import {
  DivisionCode,
  DivisionCodeFormValues,
  GeneralLedgerAccount,
} from "@/types/apps/accounting";
import { TChoiceProps } from "@/types";
import { useFormStyles } from "@/styles/FormStyles";
import axios from "@/lib/AxiosConfig";
import { APIError } from "@/types/server";
import { divisionCodeSchema } from "@/utils/apps/accounting/schema";
import { SelectInput } from "@/components/ui/fields/SelectInput";
import { statusChoices } from "@/lib/utils";
import { ValidatedTextInput } from "@/components/ui/fields/TextInput";
import { ValidatedTextArea } from "@/components/ui/fields/TextArea";

type EditDCModalFormProps = {
  divisionCode: DivisionCode;
  selectGlAccountData: TChoiceProps[];
};

export function EditDCModalForm({
  divisionCode,
  selectGlAccountData,
}: EditDCModalFormProps) {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: DivisionCodeFormValues) =>
      axios.put(`/division_codes/${divisionCode.id}/`, values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["division-code-table-data"],
          })
          .then(() => {
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
              });
            divisionCodeTableStore.set("editModalOpen", false);
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

  const form = useForm<DivisionCodeFormValues>({
    validate: yupResolver(divisionCodeSchema),
    initialValues: {
      status: divisionCode.status,
      code: divisionCode.code,
      description: divisionCode.description,
      apAccount: divisionCode.apAccount || "",
      cashAccount: divisionCode.cashAccount || "",
      expenseAccount: divisionCode.expenseAccount || "",
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
              className={classes.fields}
              name="status"
              label="Status"
              placeholder="Status"
              variant="filled"
              onMouseLeave={() => {
                form.setFieldValue("status", form.values.status);
              }}
            />
            <ValidatedTextInput<DivisionCodeFormValues>
              form={form}
              className={classes.fields}
              name="code"
              label="Code"
              placeholder="Code"
              variant="filled"
              withAsterisk
            />
          </SimpleGrid>
          <ValidatedTextArea<DivisionCodeFormValues>
            form={form}
            className={classes.fields}
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
              className={classes.fields}
              name="apAccount"
              label="AP Account"
              placeholder="AP Account"
              variant="filled"
              clearable
            />
            <SelectInput<DivisionCodeFormValues>
              form={form}
              data={selectGlAccountData}
              className={classes.fields}
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
            className={classes.fields}
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

export function EditDCModal(): React.ReactElement {
  const [showEditModal, setShowEditModal] =
    divisionCodeTableStore.use("editModalOpen");
  const [divisionCode] = divisionCodeTableStore.use("selectedRecord");
  const queryClient = useQueryClient();

  const { data: glAccountData } = useQuery({
    queryKey: "gl-account-data",
    queryFn: () => getGLAccounts(),
    enabled: showEditModal,
    initialData: () => queryClient.getQueryData("gl-account"),
    staleTime: Infinity,
  });

  const selectGlAccountData =
    glAccountData?.map((glAccount: GeneralLedgerAccount) => ({
      value: glAccount.id,
      label: glAccount.accountNumber,
    })) || [];

  return (
    <Modal.Root opened={showEditModal} onClose={() => setShowEditModal(false)}>
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Edit Division Code</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <Suspense fallback={<Skeleton height={400} />}>
            {divisionCode && (
              <EditDCModalForm
                divisionCode={divisionCode}
                selectGlAccountData={selectGlAccountData}
              />
            )}
          </Suspense>
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
