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

import { Button, Menu } from "@mantine/core";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faChevronDown } from "@fortawesome/pro-solid-svg-icons";
import React from "react";

interface ActionMenuProps<T> {
  store: any;
  data: T | null;
}

export function MontaTableActionMenu<T extends Record<string, any>>({
  store,
  data,
}: ActionMenuProps<T>) {
  return (
    <Menu
      width="10%"
      shadow="md"
      withArrow
      offset={5}
      transitionProps={{
        transition: "pop",
        duration: 150,
      }}
    >
      <Menu.Target>
        <Button
          variant="light"
          color="gray"
          size="xs"
          rightIcon={<FontAwesomeIcon icon={faChevronDown} size="sm" />}
        >
          Actions
        </Button>
      </Menu.Target>
      <Menu.Dropdown>
        <Menu.Item
          onClick={() => {
            store.set("selectedRecord", data);
            store.set("viewModalOpen", true);
          }}
        >
          View
        </Menu.Item>
        <Menu.Item
          onClick={() => {
            store.set("selectedRecord", data);
            store.set("editModalOpen", true);
          }}
        >
          Edit
        </Menu.Item>
        <Menu.Item
          color="red"
          onClick={() => {
            store.set("selectedRecord", data);
            store.set("deleteModalOpen", true);
          }}
        >
          Delete
        </Menu.Item>
      </Menu.Dropdown>
    </Menu>
  );
}
