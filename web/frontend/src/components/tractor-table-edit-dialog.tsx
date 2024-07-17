/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatToUserTimezone } from "@/lib/date";
import { cn } from "@/lib/utils";
import { tractorSchema } from "@/lib/validations/EquipmentSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  TractorFormValues as FormValues,
  Tractor,
} from "@/types/equipment";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { TractorForm } from "./tractor-table-dialog";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "./ui/sheet";

export function TractorEditForm({
  tractor,
  onOpenChange,
}: {
  tractor: Tractor;
  onOpenChange: (open: boolean) => void;
}) {
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(tractorSchema),
    defaultValues: tractor,
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/tractors/${tractor.id}/`,
    successMessage: "Tractor updated successfully.",
    queryKeysToInvalidate: "tractors",
    closeModal: true,
    reset,
    errorMessage: "Failed to update existing tractor.",
  });

  const onSubmit = (values: FormValues) => mutation.mutate(values);

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="flex h-full flex-col overflow-y-auto"
    >
      <TractorForm control={control} />
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
          Save
        </Button>
      </SheetFooter>
    </form>
  );
}

export function TractorTableEditSheet({ onOpenChange, open }: TableSheetProps) {
  const [tractor] = useTableStore.use("currentRecord") as Tractor[];

  if (!tractor) return null;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle className="flex">
            <span>{tractor.code}</span>
            <Badge className="ml-5" variant="purple">
              {tractor.id}
            </Badge>
          </SheetTitle>
          <SheetDescription>
            Last updated on {formatToUserTimezone(tractor.updatedAt)}
          </SheetDescription>
        </SheetHeader>
        <TractorEditForm tractor={tractor} onOpenChange={onOpenChange} />
      </SheetContent>
    </Sheet>
  );
}
