import {
  DataNotFound,
  StatusBadge,
} from "@/components/common/table/data-table-components";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn, truncateText } from "@/lib/utils";
import { useCustomerFormStore } from "@/stores/CustomerStore";
import { useTableStore } from "@/stores/TableStore";
import { type Customer } from "@/types/customer";
import {
  faPerson,
  faRoadCircleXmark,
} from "@fortawesome/pro-duotone-svg-icons";
import { Row } from "@tanstack/react-table";
import React from "react";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "./ui/tooltip";

const daysOfWeek = [
  "Monday", // 0
  "Tuesday", // 1
  "Wednesday", // 2
  "Thursday", // 3
  "Friday", // 4
  "Saturday", // 5
  "Sunday", // 6
];

function mapToDayOfWeek(dayOfWeek: number) {
  return daysOfWeek[dayOfWeek];
}

function CustomerContactTable({
  row,
  onClick,
}: {
  row: Row<Customer>;
  onClick: (value: string) => void;
}) {
  return (
    <div className="flex-1">
      <h2 className="scroll-m-20 pb-2 pl-3 text-2xl font-semibold tracking-tight">
        Customer Contacts
      </h2>
      <Table className="flex flex-col overflow-hidden">
        <TableHeader>
          <TableRow>
            <TableHead className="w-1/12">Status</TableHead>
            <TableHead className="w-2/12">Name</TableHead>
            <TableHead className="w-3/12">Email</TableHead>
            <TableHead className="w-2/12">Title</TableHead>
            <TableHead className="w-2/12">Phone</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {row.original.contacts && row.original.contacts.length > 0 ? (
            row.original.contacts
              .sort(
                (a, b) =>
                  new Date(b.created).getTime() - new Date(a.created).getTime(),
              )
              .map((contact) => (
                <TableRow key={contact.id} className="border-none">
                  <TableCell className="w-1/12">
                    <StatusBadge status={contact.status} />
                  </TableCell>
                  <TableCell
                    className={cn(
                      "w-2/12",
                      contact.isPayableContact && "text-red-500 font-semibold",
                    )}
                  >
                    <TooltipProvider delayDuration={100}>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span>{truncateText(contact.name, 20)}</span>
                        </TooltipTrigger>
                        <TooltipContent className="font-normal">
                          {contact.isPayableContact
                            ? "This is the payable contact"
                            : "This is not the payable contact"}
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  </TableCell>
                  <TableCell className="w-3/12">{contact.email}</TableCell>
                  <TableCell className="w-2/12">
                    {truncateText(contact.title as string, 20)}
                  </TableCell>
                  <TableCell className="w-2/12">{contact.phone}</TableCell>
                </TableRow>
              ))
          ) : (
            <DataNotFound
              message="You have not added any contacts. Add one below."
              name="contacts"
              icon={faPerson}
              onButtonClick={() => onClick("contacts")}
            />
          )}
        </TableBody>
      </Table>
    </div>
  );
}

function DeliverySlotTable({
  row,
  onClick,
}: {
  row: Row<Customer>;
  onClick: (value: string) => void;
}) {
  return (
    <div className="flex-1">
      <h2 className="scroll-m-20 pb-2 pl-3 text-2xl font-semibold tracking-tight">
        Delivery Slots
      </h2>
      <Table className="flex flex-col overflow-hidden">
        <TableHeader>
          <TableRow>
            <TableHead className="w-2/12">Day of Week</TableHead>
            <TableHead className="w-2/12">Start Time</TableHead>
            <TableHead className="w-2/12">End Time</TableHead>
            <TableHead className="w-1/12">Location</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {row.original.deliverySlots &&
          row.original.deliverySlots.length > 0 ? (
            row.original.deliverySlots
              .sort(
                (a, b) =>
                  new Date(b.created).getTime() - new Date(a.created).getTime(),
              )
              .map((deliverySlot) => (
                <TableRow key={deliverySlot.id} className="border-none ">
                  <TableCell className="w-2/12">
                    {mapToDayOfWeek(deliverySlot.dayOfWeek)}
                  </TableCell>
                  <TableCell className="w-2/12">
                    {deliverySlot.startTime}
                  </TableCell>
                  <TableCell className="w-2/12">
                    {deliverySlot.endTime}
                  </TableCell>
                  <TableCell className="w-1/12">
                    {deliverySlot.locationName}
                  </TableCell>
                </TableRow>
              ))
          ) : (
            <DataNotFound
              message="You have not added any delivery slots. Add one below."
              name="delivery slots"
              icon={faRoadCircleXmark}
              onButtonClick={() => onClick("deliverySlots")}
            />
          )}
        </TableBody>
      </Table>
    </div>
  );
}

export function CustomerTableSub({ row }: { row: Row<Customer> }) {
  const handleButtonClick = React.useCallback(
    (value: string) => {
      useTableStore.set("currentRecord", row.original);
      useTableStore.set("editSheetOpen", true);
      useCustomerFormStore.set("activeTab", value);
    },
    [row.original],
  );
  return (
    <div className="mt-5 flex border-b">
      <CustomerContactTable row={row} onClick={handleButtonClick} />
      <DeliverySlotTable row={row} onClick={handleButtonClick} />
    </div>
  );
}
