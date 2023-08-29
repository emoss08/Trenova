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

import React, { useMemo } from "react";
import { MRT_ColumnDef } from "mantine-react-table";
import { Badge, Button, Menu } from "@mantine/core";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faChevronDown } from "@fortawesome/pro-solid-svg-icons";
import { useNavigate } from "react-router-dom";
import { MontaTable } from "@/components/MontaTable";
import { CreateCommodityModal } from "@/components/commodities/CreateCommodityModal";
import { customerTableStore as store } from "@/stores/CustomerStore";
import { Customer } from "@/types/apps/customer";
import { TChoiceProps } from "@/types";
import { EditCustomerModal } from "@/components/customer/view/_partials/EditCustomerModal";

export function CustomerTable() {
  const navigate = useNavigate();
  const columns = useMemo<MRT_ColumnDef<Customer>[]>(
    () => [
      {
        id: "status",
        header: "Status",
        accessorKey: "status",
        filterFn: "equals",
        Cell: ({ cell }) => (
          <Badge
            color={cell.getValue() === "A" ? "green" : "red"}
            variant="filled"
            radius="xs"
          >
            {cell.getValue() === "A" ? "Active" : "Inactive"}
          </Badge>
        ),
        mantineFilterSelectProps: {
          data: [
            { value: "", label: "All" },
            { value: "A", label: "Active" },
            { value: "I", label: "Inactive" },
          ] satisfies ReadonlyArray<TChoiceProps>,
        },
        filterVariant: "select",
      },
      {
        accessorKey: "code",
        header: "Code",
      },
      {
        accessorKey: "name",
        header: "Name",
      },
      {
        id: "actions",
        header: "Actions",
        Cell: ({ row }) => (
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
                  navigate(`/billing/customers/view/${row.original.id}`);
                }}
              >
                View
              </Menu.Item>
              <Menu.Item
                onClick={() => {
                  navigate(`/billing/customers/edit/${row.original.id}`);
                }}
              >
                Edit
              </Menu.Item>
              <Menu.Item
                color="red"
                onClick={() => {
                  store.set("selectedRecord", row.original);
                  store.set("deleteModalOpen", true);
                }}
              >
                Delete
              </Menu.Item>
            </Menu.Dropdown>
          </Menu>
        ),
      },
    ],
    [navigate],
  );

  return (
    <MontaTable<Customer>
      store={store}
      link="/customers"
      columns={columns}
      displayDeleteModal
      tableQueryKey="customer-table-data"
      exportModelName="Customer"
      name="Customer"
    />
  );
}
