/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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

import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatToUserTimezone } from "@/lib/date";
import { cn } from "@/lib/utils";
import { tableChangeAlertSchema } from "@/lib/validations/OrganizationSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  TableChangeAlert,
  TableChangeAlertFormValues,
} from "@/types/organization";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { Badge } from "../ui/badge";
import { TableChangeAlertBody } from "./table-change-sheet";

function TableChangeEditForm({
  tableChangeAlert,
  onOpenChange,
}: {
  tableChangeAlert: TableChangeAlert;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { handleSubmit, reset, control, formState } =
    useForm<TableChangeAlertFormValues>({
      resolver: yupResolver(tableChangeAlertSchema),
      defaultValues: tableChangeAlert,
    });

  // useEffect(() => {
  //   const subscription = watch((value, { name }) => {
  //     if (name === "source" && value.source === "Database") {
  //       setValue("topicName", undefined);
  //     } else if (name === "source" && value.source === "Kafka") {
  //       setValue("tableName", undefined);
  //     }
  //   });

  //   return () => subscription.unsubscribe();
  // }, [watch, setValue]);

  console.info("Table Change Alert errors", formState.errors);

  const mutation = useCustomMutation<TableChangeAlertFormValues>(control, {
    method: "PUT",
    path: `/table-change-alerts/${tableChangeAlert.id}/`,
    successMessage: "Table Change Alert updated successfully.",
    queryKeysToInvalidate: "tableChangeAlerts",
    closeModal: true,
    reset,
    errorMessage: "Failed to update table change alert.",
  });

  const onSubmit = (values: TableChangeAlertFormValues) =>
    mutation.mutate(values);

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="flex h-full flex-col overflow-y-auto"
    >
      <TableChangeAlertBody control={control} />
      <SheetFooter className="mb-12">
        <Button
          type="reset"
          variant="secondary"
          onClick={() => onOpenChange(false)}
          className="w-full"
        >
          Cancel
        </Button>
        <Button type="submit" isLoading={mutation.isPending} className="w-full">
          Save Changes
        </Button>
      </SheetFooter>
    </form>
  );
}

export function TableChangeAlertEditSheet({
  onOpenChange,
  open,
}: TableSheetProps) {
  const [tableChangeAlert] = useTableStore.use(
    "currentRecord",
  ) as TableChangeAlert[];

  if (!tableChangeAlert) return null;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-[700px]")}>
        <SheetHeader>
          <SheetTitle className="flex">
            <span>{tableChangeAlert.name}</span>
            <Badge className="ml-5" variant="purple">
              {tableChangeAlert.id}
            </Badge>
          </SheetTitle>
          <SheetDescription>
            Last updated on {formatToUserTimezone(tableChangeAlert.updatedAt)}
          </SheetDescription>
        </SheetHeader>
        <TableChangeEditForm
          tableChangeAlert={tableChangeAlert}
          open={open}
          onOpenChange={onOpenChange}
        />
      </SheetContent>
    </Sheet>
  );
}
