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
  Skeleton,
  Textarea,
  TextInput,
} from "@mantine/core";
import React, { Suspense } from "react";
import { chargeTypeTableStore } from "@/stores/BillingStores";
import { ChargeType } from "@/types/billing";
import { useFormStyles } from "@/assets/styles/FormStyles";

type ViewChargeTypeModalFormProps = {
  chargeType: ChargeType;
};

function ViewChargeTypeModalForm({ chargeType }: ViewChargeTypeModalFormProps) {
  const { classes } = useFormStyles();

  return (
    <Box className={classes.div}>
      <Box>
        <TextInput
          value={chargeType.name}
          readOnly
          className={classes.fields}
          label="Name"
          variant="filled"
        />
        <Textarea
          value={chargeType.description || ""}
          className={classes.fields}
          label="Description"
          readOnly
          variant="filled"
        />
        <Group position="right" mt="md">
          <Button
            color="white"
            type="submit"
            onClick={() => {
              chargeTypeTableStore.set("selectedRecord", chargeType);
              chargeTypeTableStore.set("viewModalOpen", false);
              chargeTypeTableStore.set("editModalOpen", true);
            }}
            className={classes.control}
          >
            Edit Charge Type
          </Button>
        </Group>
      </Box>
    </Box>
  );
}

export function ViewChargeTypeModal(): React.ReactElement {
  const [showViewModal, setShowViewModal] =
    chargeTypeTableStore.use("viewModalOpen");
  const [chargeType] = chargeTypeTableStore.use("selectedRecord");

  return (
    <Modal.Root opened={showViewModal} onClose={() => setShowViewModal(false)}>
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>View Charge Type</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <Suspense fallback={<Skeleton />}>
            {chargeType && <ViewChargeTypeModalForm chargeType={chargeType} />}
          </Suspense>
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
