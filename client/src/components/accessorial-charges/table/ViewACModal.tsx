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
  Switch,
  Textarea,
  TextInput,
} from "@mantine/core";
import React from "react";
import { accessorialChargeTableStore } from "@/stores/BillingStores";
import { AccessorialCharge } from "@/types/billing";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { fuelMethodChoices } from "@/utils/apps/billing";

function ViewACModalForm({
  accessorialCharge,
}: {
  accessorialCharge: AccessorialCharge;
}) {
  const { classes } = useFormStyles();

  return (
    <Box className={classes.div}>
      <TextInput
        className={classes.fields}
        name="code"
        label="Code"
        description="Code for the accessorial charge."
        placeholder="Code"
        variant="filled"
        readOnly
        value={accessorialCharge.code}
      />
      <Textarea
        className={classes.fields}
        name="description"
        label="Description"
        description="Description of the accessorial charge."
        placeholder="Description"
        variant="filled"
        readOnly
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
          readOnly
          value={accessorialCharge.chargeAmount}
        />
        <Select
          data={fuelMethodChoices}
          className={classes.fields}
          name="method"
          label="Fuel Method"
          description="Method for calculating the other charge."
          placeholder="Fuel Method"
          variant="filled"
          readOnly
          value={accessorialCharge.method}
        />
        <Switch
          className={classes.fields}
          name="is_detention"
          label="Detention"
          description="Is detention charge?"
          placeholder="Detention"
          variant="filled"
          readOnly
          checked={accessorialCharge.isDetention}
        />
      </SimpleGrid>
      <Group position="right" mt="md">
        <Button
          color="white"
          type="submit"
          onClick={() => {
            accessorialChargeTableStore.set(
              "selectedRecord",
              accessorialCharge,
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
  );
}

export function ViewACModal(): React.ReactElement {
  const [showViewModal, setShowViewModal] =
    accessorialChargeTableStore.use("viewModalOpen");
  const [accessorialCharge] = accessorialChargeTableStore.use("selectedRecord");

  return (
    <Modal.Root opened={showViewModal} onClose={() => setShowViewModal(false)}>
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>View Gl Account</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          {accessorialCharge && (
            <ViewACModalForm accessorialCharge={accessorialCharge} />
          )}
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
