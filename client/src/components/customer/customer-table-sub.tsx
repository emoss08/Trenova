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

import { BoolStatusBadge } from "@/components/common/table/data-table";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { truncateText } from "@/lib/utils";
import { useCustomerFormStore } from "@/stores/CustomerStore";
import { useTableStore } from "@/stores/TableStore";
import { Customer } from "@/types/customer";
import { Row } from "@tanstack/react-table";
import React from "react";

const daysOfWeek = [
  "Sunday",
  "Monday",
  "Tuesday",
  "Wednesday",
  "Thursday",
  "Friday",
  "Saturday",
];

function mapToDayOfWeek(dayOfWeek: number) {
  return daysOfWeek[dayOfWeek];
}

export function CustomerTableSub({ row }: { row: Row<Customer> }) {
  const handleCreateContact = React.useCallback(
    (value: string) => {
      useTableStore.set("currentRecord", row.original);
      useTableStore.set("editSheetOpen", true);
      useCustomerFormStore.set("activeTab", value);
    },
    [row.original],
  );

  return (
    <div className="flex border-b mt-7">
      <div className="flex-1">
        <h2 className="scroll-m-20 pb-2 pl-5 text-2xl font-semibold tracking-tight">
          Customer Contacts
        </h2>
        <Table className="flex flex-col overflow-hidden">
          <TableHeader>
            <TableRow>
              <TableHead className="w-1/12">Active?</TableHead>
              <TableHead className="w-2/12">Name</TableHead>
              <TableHead className="w-3/12">Email</TableHead>
              <TableHead className="w-2/12">Title</TableHead>
              <TableHead className="w-2/12">Phone</TableHead>
              <TableHead className="w-2/12">Payable Contact?</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {row.original.contacts && row.original.contacts.length > 0 ? (
              row.original.contacts
                .sort(
                  (a, b) =>
                    new Date(b.created).getTime() -
                    new Date(a.created).getTime(),
                )
                .map((contact) => (
                  <TableRow key={contact.id}>
                    <TableCell className="w-1/12">
                      <BoolStatusBadge status={contact.isActive} />
                    </TableCell>
                    <TableCell className="w-2/12">
                      {truncateText(contact.name, 20)}
                    </TableCell>
                    <TableCell className="w-3/12">{contact.email}</TableCell>
                    <TableCell className="w-2/12">{contact.title}</TableCell>
                    <TableCell className="w-2/12">{contact.phone}</TableCell>
                    <TableCell className="w-2/12">
                      <BoolStatusBadge status={contact.isPayableContact} />
                    </TableCell>
                  </TableRow>
                ))
            ) : (
              <div className="flex flex-col items-center justify-center my-5">
                <p className="font-semibold text-accent-foreground">
                  No contacts found
                </p>
                <Button
                  className="mt-2"
                  size="xs"
                  onClick={() => handleCreateContact("contacts")}
                >
                  Add Contact
                </Button>
              </div>
            )}
          </TableBody>
        </Table>
      </div>

      <div className="flex-1">
        <h2 className="scroll-m-20 pb-2 pl-5 text-2xl font-semibold tracking-tight">
          Delivery Slots
        </h2>
        <Table className="flex flex-col overflow-hidden">
          <TableHeader>
            <TableRow>
              <TableHead className="w-1/12">Day of Week</TableHead>
              <TableHead className="w-2/12">Start & End Time</TableHead>
              <TableHead className="w-2/12">Location</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {row.original.deliverySlots &&
            row.original.deliverySlots.length > 0 ? (
              row.original.deliverySlots
                .sort(
                  (a, b) =>
                    new Date(b.created).getTime() -
                    new Date(a.created).getTime(),
                )
                .map((deliverySlot) => (
                  <TableRow key={deliverySlot.id}>
                    <TableCell className="w-1/12">
                      {mapToDayOfWeek(deliverySlot.dayOfWeek)}
                    </TableCell>
                    <TableCell className="w-2/12">
                      {deliverySlot.startTime} - {deliverySlot.endTime}
                    </TableCell>
                    <TableCell className="w-2/12">
                      {deliverySlot.locationName}
                    </TableCell>
                  </TableRow>
                ))
            ) : (
              <div className="flex flex-col items-center justify-center my-5">
                <p className="font-semibold text-accent-foreground">
                  No delivery slots found
                </p>
                <Button
                  className="mt-2"
                  size="xs"
                  onClick={() => handleCreateContact("deliverySlots")}
                >
                  Add Delivery Slot
                </Button>
              </div>
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
