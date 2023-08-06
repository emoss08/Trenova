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
  Select,
  SimpleGrid,
  Textarea,
  TextInput,
} from "@mantine/core";
import { useFormStyles } from "@/styles/FormStyles";
import { Commodity } from "@/types/apps/commodities";
import { unitOfMeasureChoices } from "@/utils/apps/commodities";
import { TChoiceProps } from "@/types";
import { yesAndNoChoices } from "@/lib/utils";
import { commodityTableStore } from "@/stores/CommodityStore";

type Props = {
  commodity: Commodity;
  selectHazmatData: TChoiceProps[];
};

export function ViewCommodityModalForm({ commodity, selectHazmatData }: Props) {
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
            name="min_temp"
            label="Min Temp"
            placeholder="Min Temp"
            readOnly
            variant="filled"
            value={commodity.min_temp || ""}
          />
          <TextInput
            className={classes.fields}
            name="max_temp"
            label="Max Temp"
            placeholder="Max Temp"
            readOnly
            variant="filled"
            value={commodity.max_temp || ""}
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
            name="is_hazmat"
            label="Is Hazmat"
            placeholder="Is Hazmat"
            variant="filled"
            value={commodity.is_hazmat || ""}
            readOnly
            withAsterisk
          />
        </SimpleGrid>
        <Select
          className={classes.fields}
          data={unitOfMeasureChoices}
          name="unit_of_measure"
          placeholder="Unit of Measure"
          label="Unit of Measure"
          value={commodity.is_hazmat || ""}
          readOnly
          variant="filled"
        />
        <Group position="right" mt="md">
          <Button
            color="white"
            type="submit"
            className={classes.control}
            onClick={() => {
              commodityTableStore.set("selectedRecord", commodity);
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
