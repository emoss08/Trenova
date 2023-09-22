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

import React, { Suspense } from "react";
import {
  Box,
  Button,
  Divider,
  Group,
  Modal,
  SimpleGrid,
  Skeleton,
  Text,
  Textarea,
} from "@mantine/core";
import { EquipmentType } from "@/types/equipment";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { equipmentClassChoices } from "@/helpers/choices";
import { yesAndNoChoicesBoolean } from "@/helpers/constants";
import { ViewSelectInput } from "../common/fields/SelectInput";
import { ViewTextInput } from "@/components/common/fields/TextInput";
import { useEquipTypeTableStore as store } from "@/stores/EquipmentStore";

function ModalBody({ equipType }: { equipType: EquipmentType }) {
  const { classes } = useFormStyles();

  return (
    <>
      <Box className={classes.div}>
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "lg", cols: 1 }]}>
          <ViewTextInput
            label="Name"
            value={equipType.name}
            placeholder="Name"
            description="Unique name for the equipment type."
          />
          <ViewTextInput
            label="Cost Per Mile"
            value={equipType.costPerMile}
            className={classes.fields}
            placeholder="Cost Per Mile"
            description="Cost per mile to operate the equipment type."
          />
        </SimpleGrid>
        <Textarea
          className={classes.fields}
          value={equipType.description || ""}
          label="Description"
          description="Description of the equipment type."
          placeholder="Description"
          variant="filled"
          readOnly
        />
      </Box>
      <Box className={classes.div}>
        <div
          style={{
            textAlign: "center",
            display: "flex",
            justifyContent: "center",
            alignItems: "center",
            flexDirection: "column",
          }}
        >
          <Text fz="lg" className={classes.text}>
            Equipment Type Details
          </Text>
        </div>
        <Divider my={10} />

        <SimpleGrid cols={3} breakpoints={[{ maxWidth: "lg", cols: 1 }]}>
          <ViewSelectInput
            data={equipmentClassChoices}
            value={equipType.equipmentTypeDetails.equipmentClass}
            label="Equipment Class"
            placeholder="Equipment Class"
            description="Equipment Class associated with the equipment type."
          />
          <ViewTextInput
            value={equipType.equipmentTypeDetails.fixedCost}
            label="Fixed Cost"
            placeholder="Fixed Cost"
            description="Fixed cost to operate the equipment type."
          />
          <ViewTextInput
            value={equipType.equipmentTypeDetails.variableCost}
            label="Variable Cost"
            placeholder="Variable Cost"
            description="Variable cost to operate the equipment type."
          />
          <ViewTextInput
            value={equipType.equipmentTypeDetails.height}
            label="Height"
            placeholder="Height"
            description="Height of the equipment type."
          />
          <ViewTextInput
            value={equipType.equipmentTypeDetails.weight}
            label="Weight"
            placeholder="Weight"
            description="Weight of the equipment type."
          />
          <ViewTextInput
            value={equipType.equipmentTypeDetails.length}
            label="Length"
            placeholder="Length"
            description="Length of the equipment type."
          />
          <ViewTextInput
            value={equipType.equipmentTypeDetails.width}
            label="Width"
            placeholder="Width"
            description="Current width of the equipment type."
          />
          <ViewTextInput
            value={equipType.equipmentTypeDetails.idlingFuelUsage}
            label="Idling Fuel Usage"
            placeholder="Idling Fuel Usage"
            description="Idling fuel usage of the equipment type."
          />
          <ViewSelectInput
            data={yesAndNoChoicesBoolean}
            value={equipType.equipmentTypeDetails.exemptFromTolls}
            label="Exempt From Tolls"
            placeholder="Exempt From Tolls"
            description="Exempt from tolls of the equipment type."
          />
        </SimpleGrid>
        <Group position="right" mt="md">
          <Button
            color="white"
            type="submit"
            className={classes.control}
            onClick={() => {
              store.set("viewModalOpen", false);
              store.set("editModalOpen", true);
            }}
          >
            Edit Equipment Type
          </Button>
        </Group>
      </Box>
    </>
  );
}

export function ViewEquipmentTypeModal() {
  const [equipType] = store.use("selectedRecord");
  const [showViewModal, setShowViewModal] = store.use("viewModalOpen");

  return (
    <Modal.Root
      opened={showViewModal}
      onClose={() => setShowViewModal(false)}
      size="lg"
    >
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>View Equipment Type</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <Suspense fallback={<Skeleton height={400} />}>
            {equipType && <ModalBody equipType={equipType} />}
          </Suspense>
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
