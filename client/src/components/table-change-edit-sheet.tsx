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
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { TableChangeAlertForm } from "./table-change-sheet";
import { Badge } from "./ui/badge";
function TableChangeEditForm({
  tableChangeAlert,
  open,
  onOpenChange,
}: {
  tableChangeAlert: TableChangeAlert;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { handleSubmit, reset, control, watch, setValue } =
    useForm<TableChangeAlertFormValues>({
      resolver: yupResolver(tableChangeAlertSchema),
      defaultValues: tableChangeAlert,
    });

  useEffect(() => {
    const subscription = watch((value, { name }) => {
      if (name === "source" && value.source === "Database") {
        setValue("topicName", undefined);
      } else if (name === "source" && value.source === "Kafka") {
        setValue("tableName", undefined);
      }
    });

    return () => subscription.unsubscribe();
  }, [watch, setValue]);

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
      <TableChangeAlertForm control={control} open={open} watch={watch} />
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
