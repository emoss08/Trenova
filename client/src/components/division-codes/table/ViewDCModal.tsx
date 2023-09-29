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

import { useQuery, useQueryClient } from "react-query";
import {
  Box,
  Button,
  Group,
  Modal,
  Select,
  SimpleGrid,
  Skeleton,
  Textarea,
  TextInput,
} from "@mantine/core";
import React, { Suspense } from "react";
import { divisionCodeTableStore } from "@/stores/AccountingStores";
import { getGLAccounts } from "@/services/AccountingRequestService";
import { DivisionCode, GeneralLedgerAccount } from "@/types/accounting";
import { TChoiceProps } from "@/types";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { statusChoices } from "@/lib/constants";

type ViewDCModalFormProps = {
  divisionCode: DivisionCode;
  selectGlAccountData: TChoiceProps[];
};

function ViewDCModalForm({
  divisionCode,
  selectGlAccountData,
}: ViewDCModalFormProps) {
  const { classes } = useFormStyles();

  return (
    <Box className={classes.div}>
      <Box>
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
          <Select
            data={statusChoices}
            className={classes.fields}
            readOnly
            value={divisionCode.status}
            label="Status"
            variant="filled"
          />
          <TextInput
            value={divisionCode.code}
            readOnly
            className={classes.fields}
            label="Code"
            variant="filled"
          />
        </SimpleGrid>
        <Textarea
          value={divisionCode.description}
          className={classes.fields}
          label="Description"
          readOnly
          variant="filled"
        />
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
          <Select
            data={selectGlAccountData}
            value={divisionCode.apAccount || ""}
            readOnly
            label="AP Account"
            className={classes.fields}
            variant="filled"
          />
          <Select
            data={selectGlAccountData}
            value={divisionCode.cashAccount || ""}
            readOnly
            label="Cash Account"
            className={classes.fields}
            variant="filled"
          />
        </SimpleGrid>
        <Select
          data={selectGlAccountData}
          value={divisionCode.expenseAccount || ""}
          readOnly
          label="Expense Account"
          className={classes.fields}
          variant="filled"
        />
        <Group position="right" mt="md">
          <Button
            color="white"
            type="submit"
            className={classes.control}
            onClick={() => {
              divisionCodeTableStore.set("selectedRecord", divisionCode);
              divisionCodeTableStore.set("viewModalOpen", false);
              divisionCodeTableStore.set("editModalOpen", true);
            }}
          >
            Edit Division Code
          </Button>
        </Group>
      </Box>
    </Box>
  );
}

export function ViewDCModal(): React.ReactElement {
  const [showViewModal, setShowViewModal] =
    divisionCodeTableStore.use("viewModalOpen");
  const [divisionCode] = divisionCodeTableStore.use("selectedRecord");
  const queryClient = useQueryClient();

  const { data: glAccountData } = useQuery({
    queryKey: "gl-account-data",
    queryFn: () => getGLAccounts(),
    enabled: showViewModal,
    initialData: () => queryClient.getQueryData("gl-account"),
    staleTime: Infinity,
  });

  const selectGlAccountData =
    glAccountData?.map((glAccount: GeneralLedgerAccount) => ({
      value: glAccount.id,
      label: glAccount.accountNumber,
    })) || [];

  return (
    <Modal.Root opened={showViewModal} onClose={() => setShowViewModal(false)}>
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>View Division Code</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <Suspense fallback={<Skeleton height={200} />}>
            {divisionCode && (
              <ViewDCModalForm
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
