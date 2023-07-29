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
  SimpleGrid,
  Select,
  TextInput,
  Textarea,
  Button,
  Group,
} from "@mantine/core";
import { revenueCodeTableStore } from "@/stores/AccountingStores";
import { TChoiceProps } from "@/types";
import { useFormStyles } from "@/styles/FormStyles";

type Props = {
  revenueCode: RevenueCode;
  selectGlAccountData: TChoiceProps[];
};

export const ViewRCModalForm: React.FC<Props> = ({
  revenueCode,
  selectGlAccountData,
}) => {
  const { classes } = useFormStyles();

  return (
    <>
      <Box className={classes.div}>
        <Box>
          <TextInput
            value={revenueCode.code}
            readOnly
            className={classes.fields}
            label="Code"
            variant="filled"
          />
          <Textarea
            value={revenueCode.description}
            className={classes.fields}
            label="Description"
            readOnly
            variant="filled"
          />
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <Select
              data={selectGlAccountData}
              value={revenueCode.expense_account || ""}
              readOnly
              label="AP Account"
              className={classes.fields}
              variant="filled"
            />
            <Select
              data={selectGlAccountData}
              value={revenueCode.revenue_account || ""}
              readOnly
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
