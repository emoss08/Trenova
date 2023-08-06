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

import React from "react";
import {
  Box,
  SimpleGrid,
  Select,
  TextInput,
  Textarea,
  Button,
  Group,
} from "@mantine/core";
import { GeneralLedgerAccount } from "@/types/apps/accounting";
import { statusChoices } from "@/lib/utils";
import {
  accountClassificationChoices,
  accountSubTypeChoices,
  accountTypeChoices,
  cashFlowTypeChoices,
} from "@/utils/apps/accounting";
import { generalLedgerTableStore } from "@/stores/AccountingStores";
import { useFormStyles } from "@/styles/FormStyles";

type Props = {
  glAccount: GeneralLedgerAccount;
};

export const ViewGLAccountModalForm: React.FC<Props> = ({ glAccount }) => {
  const { classes } = useFormStyles();

  return (
    <Box className={classes.div}>
      <Box>
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
            value={glAccount.account_number}
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
            value={glAccount.account_type}
            readOnly
            label="Account Type"
            className={classes.fields}
            variant="filled"
          />
          <Select
            data={cashFlowTypeChoices}
            value={glAccount.cash_flow_type}
            readOnly
            label="Cash Flow Type"
            className={classes.fields}
            variant="filled"
          />
        </SimpleGrid>
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
          <Select
            data={accountSubTypeChoices}
            value={glAccount.account_sub_type}
            readOnly
            label="Account Sub Type"
            className={classes.fields}
            variant="filled"
          />
          <Select
            data={accountClassificationChoices}
            value={glAccount.account_classification}
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
              generalLedgerTableStore.set("selectedRecord", glAccount);
              generalLedgerTableStore.set("viewModalOpen", false);
              generalLedgerTableStore.set("editModalOpen", true);
            }}
            className={classes.control}
          >
              Edit GL Account
          </Button>
        </Group>
      </Box>
    </Box>
  );
};
