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
import { glAccountSchema } from "@/lib/validations/AccountingSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  GLAccountFormValues,
  GeneralLedgerAccount,
} from "@/types/accounting";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { GLForm } from "./general-ledger-account-table-sheet";
import { Badge } from "./ui/badge";

function GLEditForm({
  glAccount,
  open,
  onOpenChange,
}: {
  glAccount: GeneralLedgerAccount;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { handleSubmit, control, getValues, setValue } =
    useForm<GLAccountFormValues>({
      resolver: yupResolver(glAccountSchema),
      defaultValues: glAccount,
    });

  const mutation = useCustomMutation<GLAccountFormValues>(control, {
    method: "PUT",
    path: `/general-ledger-accounts/${glAccount.id}/`,
    successMessage: "General Ledger Account updated successfully.",
    queryKeysToInvalidate: ["glAccounts"],
    closeModal: true,
    errorMessage: "Failed to update general ledger account.",
  });

  const onSubmit = (values: GLAccountFormValues) => mutation.mutate(values);

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="flex h-full flex-col overflow-y-auto"
    >
      <GLForm
        control={control}
        getValues={getValues}
        setValue={setValue}
        open={open}
      />
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

export function GeneralLedgerAccountTableEditSheet({
  onOpenChange,
  open,
}: TableSheetProps) {
  const [glAccount] = useTableStore.use(
    "currentRecord",
  ) as GeneralLedgerAccount[];

  if (!glAccount) return null;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle className="flex">
            <span>{glAccount.accountNumber}</span>
            <Badge className="ml-5" variant="purple">
              {glAccount.id}
            </Badge>
          </SheetTitle>
          <SheetDescription>
            Last updated on {formatToUserTimezone(glAccount.updatedAt)}
          </SheetDescription>
        </SheetHeader>
        <GLEditForm
          glAccount={glAccount}
          open={open}
          onOpenChange={onOpenChange}
        />
      </SheetContent>
    </Sheet>
  );
}
