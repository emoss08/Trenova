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
import { TitleWithTooltip } from "@/components/ui/title-with-tooltip";
import { ShipmentFormValues } from "@/types/shipment";
import { useFormContext } from "react-hook-form";
import { useTranslation } from "react-i18next";

export default function ShipmentInformation() {
  const { t } = useTranslation("shipment.addshipment");
  const { control } = useFormContext<ShipmentFormValues>();

  return (
    <div className="rounded-md border border-border bg-card">
      <div className="flex justify-center rounded-t-md border-b border-border bg-background p-2">
        <TitleWithTooltip
          title={t("card.shipmentInfo.label")}
          tooltip={t("card.shipmentInfo.description")}
        />
      </div>
      <div className="p-4">
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-4">
          <div className="col-span-2">
            <InputField
              name="mileage"
              type="number"
              control={control}
              label={t("card.shipmentInfo.fields.mileage.label")}
              placeholder={t("card.shipmentInfo.fields.mileage.placeholder")}
              description={t("card.shipmentInfo.fields.mileage.description")}
            />
          </div>
        </div>
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-4">
          <div className="col-span-1">
            <InputField
              name="pieces"
              type="number"
              control={control}
              label={t("card.shipmentInfo.fields.pieces.label")}
              placeholder={t("card.shipmentInfo.fields.pieces.placeholder")}
              description={t("card.shipmentInfo.fields.pieces.description")}
            />
          </div>
          <div className="col-span-1">
            <InputField
              name="weight"
              type="number"
              control={control}
              label={t("card.shipmentInfo.fields.weight.label")}
              placeholder={t("card.shipmentInfo.fields.weight.placeholder")}
              description={t("card.shipmentInfo.fields.weight.description")}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
