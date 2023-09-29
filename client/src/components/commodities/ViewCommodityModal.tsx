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
  Button,
  Group,
  Modal,
  Select,
  SimpleGrid,
  Textarea,
  TextInput,
} from "@mantine/core";
import { useQuery, useQueryClient } from "react-query";
import { commodityTableStore } from "@/stores/CommodityStore";
import { getHazardousMaterials } from "@/services/CommodityRequestService";
import { Commodity, HazardousMaterial } from "@/types/commodities";
import { TChoiceProps } from "@/types";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { yesAndNoChoices } from "@/lib/constants";
import { UnitOfMeasureChoices } from "@/lib/choices";

type ViewCommodityModalFormProps = {
  commodity: Commodity;
  selectHazmatData: TChoiceProps[];
};

function ViewCommodityModalForm({
  commodity,
  selectHazmatData,
}: ViewCommodityModalFormProps) {
  const { classes } = useFormStyles();

  return (
    <Box className={classes.div}>
      <Box>
        <TextInput
          className={classes.fields}
          value={commodity.name}
          name="name"
          label="Name"
          placeholder="Name"
          readOnly
          variant="filled"
          withAsterisk
        />
        <Textarea
          className={classes.fields}
          name="description"
          label="Description"
          placeholder="Description"
          readOnly
          variant="filled"
          value={commodity.description || ""}
        />
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
          <TextInput
            className={classes.fields}
            name="minTemp"
            label="Min Temp"
            placeholder="Min Temp"
            readOnly
            variant="filled"
            value={commodity.minTemp || ""}
          />
          <TextInput
            className={classes.fields}
            name="maxTemp"
            label="Max Temp"
            placeholder="Max Temp"
            readOnly
            variant="filled"
            value={commodity.maxTemp || ""}
          />
        </SimpleGrid>
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
          <Select
            className={classes.fields}
            data={selectHazmatData || []}
            name="hazmat"
            placeholder="Hazardous Material"
            label="Hazardous Material"
            variant="filled"
            value={commodity.hazmat || ""}
            readOnly
            clearable
          />
          <Select
            className={classes.fields}
            data={yesAndNoChoices}
            name="isHazmat"
            label="Is Hazmat"
            placeholder="Is Hazmat"
            variant="filled"
            value={commodity.isHazmat || ""}
            readOnly
            withAsterisk
          />
        </SimpleGrid>
        <Select
          className={classes.fields}
          data={UnitOfMeasureChoices}
          name="unitOfMeasure"
          placeholder="Unit of Measure"
          label="Unit of Measure"
          value={commodity.unitOfMeasure || ""}
          readOnly
          variant="filled"
        />
        <Group position="right" mt="md">
          <Button
            color="white"
            type="submit"
            className={classes.control}
            onClick={() => {
              commodityTableStore.set("viewModalOpen", false);
              commodityTableStore.set("editModalOpen", true);
            }}
          >
            Edit Commodity
          </Button>
        </Group>
      </Box>
    </Box>
  );
}

export function ViewCommodityModal() {
  const [showViewModal, setShowViewModal] =
    commodityTableStore.use("viewModalOpen");
  const [commodity] = commodityTableStore.use("selectedRecord");
  const queryClient = useQueryClient();

  const { data: hazmatData } = useQuery({
    queryKey: "hazmat-data",
    queryFn: () => getHazardousMaterials(),
    enabled: showViewModal,
    initialData: () => queryClient.getQueryData("hazmat-data"),
    staleTime: Infinity,
  });

  const selectHazmatData =
    hazmatData?.map((hazardousMaterial: HazardousMaterial) => ({
      value: hazardousMaterial.id,
      label: hazardousMaterial.name,
    })) || [];

  return (
    <Modal.Root opened={showViewModal} onClose={() => setShowViewModal(false)}>
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>View Commodity</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          {commodity && (
            <ViewCommodityModalForm
              commodity={commodity}
              selectHazmatData={selectHazmatData}
            />
          )}
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
