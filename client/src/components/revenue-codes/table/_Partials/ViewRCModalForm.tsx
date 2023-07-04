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

import { RevenueCode } from "@/types/apps/accounting";
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
import { revenueCodeTableStore } from "@/stores/AccountingStores";
import { ChoiceProps } from "@/types";

type Props = {
  revenueCode: RevenueCode;
  selectGlAccountData: ChoiceProps[];
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

export const ViewRCModalForm: React.FC<Props> = ({
  revenueCode,
  selectGlAccountData,
}) => {
  const { classes } = useStyles();

  return (
    <>
      <Box className={classes.div}>
        <Box>
          <TextInput
            value={revenueCode.code}
            disabled
            className={classes.fields}
            label="Code"
            variant="filled"
          />
          <Textarea
            value={revenueCode.description}
            className={classes.fields}
            label="Description"
            disabled
            variant="filled"
          />
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <Select
              data={selectGlAccountData}
              value={revenueCode.expense_account || ""}
              disabled
              label="AP Account"
              className={classes.fields}
              variant="filled"
            />
            <Select
              data={selectGlAccountData}
              value={revenueCode.revenue_account || ""}
              disabled
              label="Cash Account"
              className={classes.fields}
              variant="filled"
            />
          </SimpleGrid>
          <Group position="right" mt="md">
            <Button
              color="white"
              type="submit"
              onClick={() => {
                revenueCodeTableStore.set("selectedRecord", revenueCode);
                revenueCodeTableStore.set("viewModalOpen", false);
                revenueCodeTableStore.set("editModalOpen", true);
              }}
              className={classes.control}
            >
              Edit Revenue Code
            </Button>
          </Group>
        </Box>
      </Box>
    </>
  );
};
