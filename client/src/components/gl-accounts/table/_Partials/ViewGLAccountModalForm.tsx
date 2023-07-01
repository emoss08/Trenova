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
  createStyles,
  rem,
  SimpleGrid,
  Select,
  TextInput,
  Textarea,
  Button,
  Group,
} from "@mantine/core";
import { statusChoices } from "@/lib/utils";
import {
  accountClassificationChoices,
  accountSubTypeChoices,
  accountTypeChoices,
  cashFlowTypeChoices,
} from "@/utils/apps/accounting";
import { generalLedgerTableStore } from "@/stores/AccountingStores";

type Props = {
  glAccount: GeneralLedgerAccount;
};

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

export const ViewGLAccountModalForm: React.FC<Props> = ({ glAccount }) => {
  const { classes } = useStyles();

  return (
    <>
      <Box className={classes.div}>
        <Box>
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <Select
              data={statusChoices}
              className={classes.fields}
              disabled
              value={glAccount.status}
              label="Status"
              variant="filled"
            />
            <TextInput
              value={glAccount.account_number}
              disabled
              className={classes.fields}
              label="Account Number"
              variant="filled"
            />
          </SimpleGrid>
          <Textarea
            value={glAccount.description}
            className={classes.fields}
            label="Description"
            disabled
            variant="filled"
          />
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <Select
              data={accountTypeChoices}
              value={glAccount.account_type}
              disabled
              label="Account Type"
              className={classes.fields}
              variant="filled"
            />
            <Select
              data={cashFlowTypeChoices}
              value={glAccount.cash_flow_type}
              disabled
              label="Cash Flow Type"
              className={classes.fields}
              variant="filled"
            />
          </SimpleGrid>
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <Select
              data={accountSubTypeChoices}
              value={glAccount.account_sub_type}
              disabled
              label="Account Sub Type"
              className={classes.fields}
              variant="filled"
            />
            <Select
              data={accountClassificationChoices}
              value={glAccount.account_classification}
              disabled
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
    </>
  );
};
