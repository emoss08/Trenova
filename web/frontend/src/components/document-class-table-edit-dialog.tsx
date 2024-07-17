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
import { documentClassSchema } from "@/lib/validations/BillingSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  DocumentClassification,
  DocumentClassificationFormValues as FormValues,
} from "@/types/billing";
import { type TableSheetProps } from "@/types/tables";

import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { DocumentClassForm } from "./document-class-table-dialog";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import {
  Credenza,
  CredenzaBody,
  CredenzaClose,
  CredenzaContent,
  CredenzaDescription,
  CredenzaFooter,
  CredenzaHeader,
  CredenzaTitle,
} from "./ui/credenza";

function DocumentClassEditForm({
  documentClass,
}: {
  documentClass: DocumentClassification;
}) {
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(documentClassSchema),
    defaultValues: documentClass,
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/document-classifications/${documentClass.id}/`,
    successMessage: "Document Classification updated successfully.",
    queryKeysToInvalidate: "documentClassifications",
    closeModal: true,
    reset,
    errorMessage: "Failed to update document classification.",
  });

  const onSubmit = (values: FormValues) => mutation.mutate(values);

  return (
    <CredenzaBody>
      <form onSubmit={handleSubmit(onSubmit)}>
        <DocumentClassForm control={control} />
        <CredenzaFooter>
          <CredenzaClose asChild>
            <Button variant="outline" type="button">
              Cancel
            </Button>
          </CredenzaClose>
          <Button type="submit" isLoading={mutation.isPending}>
            Save Changes
          </Button>
        </CredenzaFooter>
      </form>
    </CredenzaBody>
  );
}

export function DocumentClassEditDialog({
  onOpenChange,
  open,
}: TableSheetProps) {
  const [documentClass] = useTableStore.use(
    "currentRecord",
  ) as DocumentClassification[];

  if (!documentClass) return null;

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle className="flex">
            <span>{documentClass.code}</span>
            <Badge className="ml-5" variant="purple">
              {documentClass.id}
            </Badge>
          </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on {formatToUserTimezone(documentClass.updatedAt)}
        </CredenzaDescription>
        <DocumentClassEditForm documentClass={documentClass} />
      </CredenzaContent>
    </Credenza>
  );
}
