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

import { SelectInput } from "@/components/common/fields/select-input";
import { TitleWithTooltip } from "@/components/ui/title-with-tooltip";
import { useCustomers } from "@/hooks/useQueries";
import { ShipmentFormValues } from "@/types/order";
import { Control } from "react-hook-form";
import { useTranslation } from "react-i18next";

export function CustomerInformation({
  control,
}: {
  control: Control<ShipmentFormValues>;
}) {
  const { t } = useTranslation("shipment.addshipment");
  const {
    selectCustomersData,
    isError: isCustomersError,
    isLoading: isCustomersLoading,
  } = useCustomers();

  return (
    <div className="border-border bg-card rounded-md border">
      <div className="border-border bg-background flex justify-center rounded-t-md border-b p-2">
        <TitleWithTooltip
          title={t("card.customerInfo.label")}
          tooltip={t("card.customerInfo.description")}
        />
      </div>
      <div className="p-4">
        <div className="grid max-w-3xl grid-cols-1 gap-4 sm:grid-cols-4">
          <div className="col-span-2">
            <SelectInput
              name="customer"
              control={control}
              options={selectCustomersData}
              isLoading={isCustomersLoading}
              isFetchError={isCustomersError}
              rules={{ required: true }}
              label={t("card.customerInfo.fields.customer.label")}
              placeholder={t("card.customerInfo.fields.customer.placeholder")}
              description={t("card.customerInfo.fields.customer.description")}
              popoutLink="/billing/customers/"
              hasPopoutWindow
              popoutLinkLabel="Customer"
            />
          </div>
        </div>
      </div>
    </div>
  );
}
