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

import { DivisionCode } from "@/types/apps/accounting";
import React from "react";
import { Box, SimpleGrid, Select, TextInput, Textarea } from "@mantine/core";
import { statusChoices } from "@/lib/utils";
import { ChoiceProps } from "@/types";
import { useFormStyles } from "@/styles/FormStyles";

type Props = {
  divisionCode: DivisionCode;
  selectGlAccountData: ChoiceProps[];
};

export const ViewDCModalForm: React.FC<Props> = ({
  divisionCode,
  selectGlAccountData,
}) => {
  const { classes } = useFormStyles();

  return (
    <>
      <Box className={classes.div}>
        <Box>
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <Select
              data={statusChoices}
              className={classes.fields}
              disabled
              value={divisionCode.status}
              label="Status"
              variant="filled"
            />
            <TextInput
              value={divisionCode.code}
              disabled
              className={classes.fields}
              label="Code"
              variant="filled"
            />
          </SimpleGrid>
          <Textarea
            value={divisionCode.description}
            className={classes.fields}
            label="Description"
            disabled
            variant="filled"
          />
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <Select
              data={selectGlAccountData}
              value={divisionCode.ap_account || ""}
              disabled
              label="AP Account"
              className={classes.fields}
              variant="filled"
            />
            <Select
              data={selectGlAccountData}
              value={divisionCode.cash_account || ""}
              disabled
              label="Cash Account"
              className={classes.fields}
              variant="filled"
            />
          </SimpleGrid>
          <Select
            data={selectGlAccountData}
            value={divisionCode.expense_account || ""}
            disabled
            label="Expense Account"
            className={classes.fields}
            variant="filled"
          />
        </Box>
      </Box>
    </>
  );
};
