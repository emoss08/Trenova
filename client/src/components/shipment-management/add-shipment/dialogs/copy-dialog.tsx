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

import { InputField } from "@/components/common/fields/input";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FormControl } from "@/components/ui/form";
import { ShipmentFormValues } from "@/types/order";
import { Control } from "react-hook-form";
import { useTranslation } from "react-i18next";

export function CopyShipmentDialog({
  onOpenChange,
  open,
  control,
  submitForm,
  isSubmitting,
}: {
  onOpenChange: (open: boolean) => void;
  open: boolean;
  control: Control<ShipmentFormValues>;
  submitForm: () => void;
  isSubmitting: boolean;
}) {
  const { t } = useTranslation("shipment.addshipment");

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("dialog.saveCopy.title")}</DialogTitle>
        </DialogHeader>
        <DialogDescription>{t("dialog.saveCopy.content")}</DialogDescription>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="copyAmount"
            label={t("dialog.saveCopy.fields.copyAmount.label")}
            placeholder={t("dialog.saveCopy.fields.copyAmount.placeholder")}
            description={t("dialog.saveCopy.fields.copyAmount.description")}
            type="number"
          />
        </FormControl>
        <DialogFooter>
          <Button
            size="sm"
            type="submit"
            onClick={() => {
              submitForm();
            }}
            isLoading={isSubmitting}
          >
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
