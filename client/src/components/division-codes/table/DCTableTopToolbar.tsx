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

import { Box, Button, Checkbox, Flex, Popover } from "@mantine/core";
import { MRT_GlobalFilterTextInput } from "mantine-react-table";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  faFileExport,
  faFilters,
  faUserPlus,
} from "@fortawesome/pro-duotone-svg-icons";
import React from "react";
import { divisionCodeTableStore } from "@/stores/AccountingStores";

interface TopToolbarProps {
  table: any;
}

export const DCTableTopToolbar: React.FC<TopToolbarProps> = ({ table }) => {
  const [showColumnFilters, setShowColumnFilters] =
    divisionCodeTableStore.use("columnFilters");
  // const rowSelection = table.getState().rowSelection;
  //
  // const selectedRowCount = Object.keys(rowSelection).filter(
  //   (key) => rowSelection[key]
  // ).length;

  return (
    <>
      {/*{selectedRowCount > 0 && (*/}
      {/*  <Text>*/}
      {/*    {selectedRowCount} user{selectedRowCount > 1 && "s"} selected*/}
      {/*  </Text>*/}
      {/*)}*/}
      <Flex
        sx={() => ({
          borderRadius: "4px",
          flexDirection: "row",
          justifyContent: "space-between",
          padding: "24px 16px",
          "@media max-width: 768px": {
            flexDirection: "column",
          },
        })}
      >
        <Box
          sx={{
            display: "flex",
            gap: "16px",
            flexWrap: "wrap",
            flex: 1,
            justifyContent: "flex-start",
          }}
        >
          <MRT_GlobalFilterTextInput table={table} />
        </Box>

        <Flex
          gap="xs"
          align="center"
          sx={{
            flex: 1,
            justifyContent: "flex-end",
          }}
        >
          <Popover
            width={300}
            trapFocus
            position="bottom"
            withArrow
            shadow="md"
          >
            <Popover.Target>
              <Button
                color="blue"
                leftIcon={<FontAwesomeIcon icon={faFilters} />}
              >
                Filter
              </Button>
            </Popover.Target>
            <Popover.Dropdown
              sx={(theme) => ({
                background:
                  theme.colorScheme === "dark"
                    ? theme.colors.dark[7]
                    : theme.white,
              })}
            >
              <Checkbox
                label="Show/Hide Column Filters"
                onChange={(event) => {
                  setShowColumnFilters(event.target.checked);
                  table.setShowColumnFilters(event.target.checked);
                }}
                checked={showColumnFilters}
                size="sm"
              />
            </Popover.Dropdown>
          </Popover>

          <Button
            color="blue"
            leftIcon={<FontAwesomeIcon icon={faFileExport} />}
            onClick={() => divisionCodeTableStore.set("exportModalOpen", true)}
          >
            Export
          </Button>
          <Button
            color="blue"
            onClick={() => divisionCodeTableStore.set("createModalOpen", true)}
            leftIcon={<FontAwesomeIcon icon={faUserPlus} />}
          >
            Add Division Code
          </Button>
        </Flex>
      </Flex>
    </>
  );
};
