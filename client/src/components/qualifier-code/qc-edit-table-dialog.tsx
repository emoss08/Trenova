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

import { QualifierCodeForm } from "@/components/qualifier-code/qc-table-dialog";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatDate } from "@/lib/date";
import { qualifierCodeSchema } from "@/lib/validations/StopSchema";
import { useTableStore } from "@/stores/TableStore";
import {
  QualifierCodeFormValues as FormValues,
  QualifierCode,
} from "@/types/stop";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";

function QualifierCodeEditForm({
  qualifierCode,
}: {
  qualifierCode: QualifierCode;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(qualifierCodeSchema),
    defaultValues: {
      status: qualifierCode.status,
      code: qualifierCode.code,
      description: qualifierCode.description,
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/qualifier_codes/${qualifierCode.id}/`,
      successMessage: "Qualifier Code updated successfully.",
      queryKeysToInvalidate: ["qualifier-code-table-data"],
      closeModal: true,
      errorMessage: "Failed to update qualifier code",
    },
    () => setIsSubmitting(false),
  );

  const onSubmit = (values: FormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="flex flex-col h-full overflow-y-auto"
    >
      <QualifierCodeForm control={control} />
      <DialogFooter className="mt-6">
        <Button
          type="submit"
          isLoading={isSubmitting}
          loadingText="Saving Changes..."
        >
          Save
        </Button>
      </DialogFooter>
    </form>
  );
}

export function QualifierCodeEditDialog({
  onOpenChange,
  open,
}: TableSheetProps) {
  const [qualifierCode] = useTableStore.use("currentRecord") as QualifierCode[];

  if (!qualifierCode) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{qualifierCode && qualifierCode.code}</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Last updated on&nbsp;
          {qualifierCode && formatDate(qualifierCode.modified)}
        </DialogDescription>
        {qualifierCode && (
          <QualifierCodeEditForm qualifierCode={qualifierCode} />
        )}
      </DialogContent>
    </Dialog>
  );
}
