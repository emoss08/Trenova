/*
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

import { DocumentClassForm } from "@/components/document-class/document-class-table-dialog";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatDate } from "@/lib/date";
import { documentClassSchema } from "@/lib/validations/BillingSchema";
import { useTableStore } from "@/stores/TableStore";
import {
  DocumentClassification,
  DocumentClassificationFormValues as FormValues,
} from "@/types/billing";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { Button } from "../ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../ui/dialog";

function DocumentClassEditForm({
  documentClass,
}: {
  documentClass: DocumentClassification;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(documentClassSchema),
    defaultValues: {
      name: documentClass.name,
      description: documentClass.description,
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/document_classifications/${documentClass.id}/`,
      successMessage: "Document Classification updated successfully.",
      queryKeysToInvalidate: ["document-classification-table-data"],
      closeModal: true,
      errorMessage: "Failed to update document classification.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: FormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <DocumentClassForm control={control} />
      <DialogFooter className="mt-6">
        <Button
          type="submit"
          isLoading={isSubmitting}
        >
          Save
        </Button>
      </DialogFooter>
    </form>
  );
}

export function DocumentClassEditDialog({
  onOpenChange,
  open,
}: TableSheetProps) {
  const [documentClass] = useTableStore.use(
    "currentRecord",
  ) as DocumentClassification[];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{documentClass && documentClass.name}</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Last updated on {documentClass && formatDate(documentClass.modified)}
        </DialogDescription>
        {documentClass && (
          <DocumentClassEditForm documentClass={documentClass} />
        )}
      </DialogContent>
    </Dialog>
  );
}
