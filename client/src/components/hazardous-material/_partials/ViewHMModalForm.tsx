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
  Select,
  SimpleGrid,
  Textarea,
  TextInput,
} from "@mantine/core";
import React from "react";
import { statusChoices } from "@/lib/utils";
import { useFormStyles } from "@/styles/FormStyles";
import { HazardousMaterial } from "@/types/apps/commodities";
import { hazardousMaterialTableStore } from "@/stores/CommodityStore";
import {
  hazardousClassChoices,
  packingGroupChoices,
} from "@/utils/apps/commodities";

type Props = {
  hazardousMaterial: HazardousMaterial;
};

export function ViewHMModalForm({ hazardousMaterial }: Props) {
  const { classes } = useFormStyles();

  return (
    <Box className={classes.div}>
      <Box>
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
          <Select
            data={statusChoices}
            className={classes.fields}
            value={hazardousMaterial.status}
            name="status"
            label="Status"
            placeholder="Status"
            variant="filled"
            readOnly
          />
          <TextInput
            className={classes.fields}
            value={hazardousMaterial.name}
            name="name"
            label="Name"
            placeholder="Name"
            variant="filled"
            readOnly
          />
        </SimpleGrid>
        <Textarea
          className={classes.fields}
          value={hazardousMaterial.description || ""}
          name="description"
          label="Description"
          placeholder="Description"
          variant="filled"
          readOnly
        />
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
          <Select
            data={hazardousClassChoices}
            className={classes.fields}
            value={hazardousMaterial.hazard_class}
            name="hazard_class"
            label="Hazard Class"
            placeholder="Hazard Class"
            variant="filled"
            readOnly
          />
          <Select
            data={packingGroupChoices}
            className={classes.fields}
            value={hazardousMaterial.packing_group || ""}
            name="packing_group"
            label="Packing Group"
            placeholder="Packing Group"
            variant="filled"
            readOnly
          />
        </SimpleGrid>
        <TextInput
          className={classes.fields}
          value={hazardousMaterial.erg_number || ""}
          name="erg_number"
          label="ERG Number"
          placeholder="ERG Number"
          variant="filled"
          readOnly
        />
        <TextInput
          className={classes.fields}
          value={hazardousMaterial.proper_shipping_name || ""}
          name="proper_shipping_name"
          label="Proper Shipping Name"
          placeholder="Proper Shipping Name"
          variant="filled"
          readOnly
        />
        <Group position="right" mt="md">
          <Button
            color="white"
            type="submit"
            onClick={() => {
              hazardousMaterialTableStore.set(
                "selectedRecord",
                hazardousMaterial
              );
              hazardousMaterialTableStore.set("viewModalOpen", false);
              hazardousMaterialTableStore.set("editModalOpen", true);
            }}
            className={classes.control}
          >
              Edit Hazardous Material
          </Button>
        </Group>
      </Box>
    </Box>
  );
}
