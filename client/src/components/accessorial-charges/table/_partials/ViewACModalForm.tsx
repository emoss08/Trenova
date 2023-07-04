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
  Switch,
} from "@mantine/core";
import { useFormStyles } from "@/styles/FormStyles";
import { AccessorialCharge } from "@/types/apps/billing";
import { fuelMethodChoices } from "@/utils/apps/billing";
import { accessorialChargeTableStore } from "@/stores/BillingStores";

type Props = {
  accessorialCharge: AccessorialCharge;
};

export const ViewACModalForm: React.FC<Props> = ({ accessorialCharge }) => {
  const { classes } = useFormStyles();

  return (
    <>
      <Box className={classes.div}>
        <Box>
          <TextInput
            className={classes.fields}
            name="code"
            label="Code"
            description="Code for the accessorial charge."
            placeholder="Code"
            variant="filled"
            disabled
            value={accessorialCharge.code}
          />
          <Textarea
            className={classes.fields}
            name="description"
            label="Description"
            description="Description of the accessorial charge."
            placeholder="Description"
            variant="filled"
            disabled
            value={accessorialCharge.description || ""}
          />
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <TextInput
              className={classes.fields}
              name="charge_amount"
              label="Charge Amount"
              placeholder="Charge Amount"
              description="Charge amount for the accessorial charge."
              variant="filled"
              disabled
              value={accessorialCharge.charge_amount}
            />
            <Select
              data={fuelMethodChoices}
              className={classes.fields}
              name="method"
              label="Fuel Method"
              description="Method for calculating the other charge."
              placeholder="Fuel Method"
              variant="filled"
              disabled
              value={accessorialCharge.method}
            />
            <Switch
              className={classes.fields}
              name="is_detention"
              label="Detention"
              description="Is detention charge?"
              placeholder="Detention"
              variant="filled"
              disabled
              checked={accessorialCharge.is_detention}
            />
          </SimpleGrid>
          <Group position="right" mt="md">
            <Button
              color="white"
              type="submit"
              onClick={() => {
                accessorialChargeTableStore.set(
                  "selectedRecord",
                  accessorialCharge
                );
                accessorialChargeTableStore.set("viewModalOpen", false);
                accessorialChargeTableStore.set("editModalOpen", true);
              }}
              className={classes.control}
            >
              Edit Accessorial Charge
            </Button>
          </Group>
        </Box>
      </Box>
    </>
  );
};
