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

import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { cn } from "@/lib/utils";
import { TableSheetProps } from "@/types/tables";
import {
  accountClassificationChoices,
  accountSubTypeChoices,
  accountTypeChoices,
  cashFlowTypeChoices,
  statusChoices,
} from "@/lib/choices";
import { FileField, InputField } from "../ui/input";
import { TChoiceProps } from "@/types";
import { useGLAccounts } from "@/hooks/useGLAccounts";
import { TextareaField } from "../ui/textarea";
import { SelectInput, ComboboxDemo } from "../ui/select";

function GLForm({
  glAccounts,
  isGLAccountsError,
  isGLAccountsLoading,
}: {
  glAccounts: TChoiceProps[];
  isGLAccountsLoading: boolean;
  isGLAccountsError: boolean;
}) {
  const frameworks = [
    {
      value: "next.js",
      label: "Next.js",
    },
    {
      value: "sveltekit",
      label: "SvelteKit",
    },
    {
      value: "nuxt.js",
      label: "Nuxt.js",
    },
    {
      value: "remix",
      label: "Remix",
    },
    {
      value: "astro",
      label: "Astro",
    },
  ];

  return (
    <div>
      <div className="grid grid-cols-3 gap-6 my-4">
        <div className="grid w-full max-w-sm items-center gap-1.5">
          {/* <SelectInput
            label="Status"
            data={statusChoices}
            placeholder="Select Status"
            description="Status of the Account"
            withAsterisk
          /> */}
          <ComboboxDemo frameworks={frameworks} />
        </div>
        <div className="grid w-full max-w-sm items-center gap-1.5">
          <InputField
            label="Account Number"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Account Number"
            autoComplete="accountNumber"
            description="The account number of the account"
            withAsterisk
            // disabled={isLoading}
            // error={errors?.username?.message}
            // {...register("username")}
          />
        </div>
      </div>
      <TextareaField
        label="Description"
        placeholder="Description"
        description="The description of the account"
        withAsterisk
      />
      <div className="grid grid-cols-3 gap-6 my-4">
        <div className="grid w-full max-w-sm items-center gap-1.5">
          <SelectInput
            label="Account Type"
            data={accountTypeChoices}
            placeholder="Select Account Type"
            description="The Account Type of GL Account"
            withAsterisk
          />
        </div>
        <div className="grid w-full max-w-sm items-center gap-1.5">
          <SelectInput
            label="Cash Flow Type"
            data={cashFlowTypeChoices}
            placeholder="Select Cash Flow Type"
            description="The Cash Flow Type of the GL Account"
            withAsterisk
          />
        </div>
        <div className="grid w-full max-w-sm items-center gap-1.5">
          <SelectInput
            label="Account Sub Type"
            data={accountSubTypeChoices}
            placeholder="Select Account Sub Type"
            description="The Account Sub Type of the GL Account"
            withAsterisk
          />
        </div>
        <div className="grid w-full max-w-sm items-center gap-1.5">
          <SelectInput
            label="Account Classification"
            data={accountClassificationChoices}
            placeholder="Select Account Classification"
            description="The Account Classification of the GL Account"
            withAsterisk
          />
        </div>
        <div className="grid w-full max-w-sm items-center gap-1.5">
          <SelectInput
            label="Account Classification"
            data={accountClassificationChoices}
            placeholder="Select Account Classification"
            description="The Account Classification of the GL Account"
            withAsterisk
          />
        </div>
        <div className="grid w-full max-w-sm items-center gap-1.5">
          <SelectInput
            label="Parent Account"
            data={glAccounts}
            isLoading={isGLAccountsLoading}
            isError={isGLAccountsError}
            placeholder="Select Parent Account"
            description="Parent account for hierarchical accounting"
            withAsterisk
          />
        </div>
        <div className="grid w-full max-w-sm items-center gap-1.5">
          <FileField
            label="Parent Account"
            placeholder="Select Parent Account"
            description="Parent account for hierarchical accounting"
            withAsterisk
          />
        </div>
      </div>
    </div>
  );
}

export function GLTableSheet({ onOpenChange, open }: TableSheetProps) {
  const {
    selectGLAccounts,
    isError: glAccountsError,
    isLoading: glAccountsLoading,
  } = useGLAccounts(open);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-1/2")}>
        <SheetHeader>
          <SheetTitle>Add New GL Account</SheetTitle>
          <SheetDescription>
            <SheetDescription>
              Use this form to add a new general ledger account to the system.
            </SheetDescription>
          </SheetDescription>
        </SheetHeader>
        <GLForm
          glAccounts={selectGLAccounts}
          isGLAccountsLoading={glAccountsLoading}
          isGLAccountsError={glAccountsError}
        />
        <SheetFooter>
          <SheetClose asChild>
            <Button type="submit">Save changes</Button>
          </SheetClose>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
