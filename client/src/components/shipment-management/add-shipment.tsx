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
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { statusChoices } from "@/lib/choices";
import { TabsContent } from "@radix-ui/react-tabs";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";

function ShipmentGeneralForm() {
  const { t } = useTranslation(["admin.accountingcontrol", "common"]);
  const { control } = useForm();
  return (
    <div className="sm:p8 px-4 py-6">
      <div className="grid max-w-3xl grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-6">
        <div className="col-span-3">
          <SelectInput
            name="journalEntryCriteria"
            control={control}
            options={statusChoices}
            rules={{ required: true }}
            label={t("fields.journalEntryCriteria.label")}
            placeholder={t("fields.journalEntryCriteria.placeholder")}
            description={t("fields.journalEntryCriteria.description")}
          />
        </div>
      </div>
    </div>
  );
}

export default function AddShipment() {
  return (
    <div className="bg-card border-border m-4 border sm:rounded-xl md:col-span-2">
      <Tabs defaultValue="general">
        <TabsList>
          <TabsTrigger value="general">General Information</TabsTrigger>
          <TabsTrigger value="billing">Billing Information</TabsTrigger>
        </TabsList>
        <TabsContent value="general">
          <ShipmentGeneralForm />
        </TabsContent>
        <TabsContent value="billing">
          Billing information is a work in progress
        </TabsContent>
      </Tabs>
    </div>
  );
}
