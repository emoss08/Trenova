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
  Select,
  SimpleGrid,
  Skeleton,
  Textarea,
  TextInput,
} from "@mantine/core";
import React, { Suspense } from "react";
import { generalLedgerTableStore as store } from "@/stores/AccountingStores";
import { GeneralLedgerAccount } from "@/types/accounting";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { statusChoices } from "@/helpers/constants";
import {
  accountClassificationChoices,
  accountSubTypeChoices,
  accountTypeChoices,
  cashFlowTypeChoices,
} from "@/helpers/choices";

type ViewGLAccountModalFormProps = {
  glAccount: GeneralLedgerAccount;
};

function ViewGLAccountModalForm({
  glAccount,
}: ViewGLAccountModalFormProps): React.ReactElement {
  const { classes } = useFormStyles();

  return (
    <Box className={classes.div}>
      <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
        <Select
          data={statusChoices}
          className={classes.fields}
          readOnly
          value={glAccount.status}
          label="Status"
          variant="filled"
        />
        <TextInput
          value={glAccount.accountNumber}
          readOnly
          className={classes.fields}
          label="Account Number"
          variant="filled"
        />
      </SimpleGrid>
      <Textarea
        value={glAccount.description}
        className={classes.fields}
        label="Description"
        readOnly
        variant="filled"
      />
      <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
        <Select
          data={accountTypeChoices}
          value={glAccount.accountType}
          readOnly
          label="Account Type"
          className={classes.fields}
          variant="filled"
        />
        <Select
          data={cashFlowTypeChoices}
          value={glAccount.cashFlowType}
          readOnly
          label="Cash Flow Type"
          className={classes.fields}
          variant="filled"
        />
      </SimpleGrid>
      <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
        <Select
          data={accountSubTypeChoices}
          value={glAccount.accountSubType}
          readOnly
          label="Account Sub Type"
          className={classes.fields}
          variant="filled"
        />
        <Select
          data={accountClassificationChoices}
          value={glAccount.accountClassification}
          readOnly
          label="Account Classification"
          className={classes.fields}
          variant="filled"
        />
      </SimpleGrid>
      <Group position="right" mt="md">
        <Button
          color="white"
          type="submit"
          onClick={() => {
            store.set("selectedRecord", glAccount);
            store.set("viewModalOpen", false);
            store.set("editModalOpen", true);
          }}
          className={classes.control}
        >
          Edit GL Account
        </Button>
      </Group>
    </Box>
  );
}

export function ViewGLAccountModal(): React.ReactElement {
  const [showViewModal, setShowViewModal] = store.use("viewModalOpen");
  const [glAccount] = store.use("selectedRecord");

  return (
    <Modal.Root opened={showViewModal} onClose={() => setShowViewModal(false)}>
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>View Gl Account</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <Suspense fallback={<Skeleton height={400} />}>
            {glAccount && <ViewGLAccountModalForm glAccount={glAccount} />}
          </Suspense>
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
