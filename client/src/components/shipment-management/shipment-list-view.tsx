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

import { ShipmentAsideMenus } from "@/components/shipment-management/map-view/shipment-aside-menu";
import { ShipmentList } from "@/components/shipment-management/shipment-list";
import { ShipmentSearchForm } from "@/types/shipment";
import { Control, UseFormSetValue, UseFormWatch } from "react-hook-form";

export function ShipmentListView({
  finalStatuses,
  progressStatuses,
  control,
  setValue,
  watch,
}: {
  finalStatuses: string[];
  progressStatuses: string[];
  control: Control<ShipmentSearchForm>;
  setValue: UseFormSetValue<ShipmentSearchForm>;
  watch: UseFormWatch<ShipmentSearchForm>;
}) {
  return (
    <div className="flex w-full space-x-10">
      <div className="w-1/4">
        <ShipmentAsideMenus
          control={control}
          setValue={setValue}
          watch={watch}
        />
      </div>
      <div className="w-3/4">
        <ShipmentList
          finalStatuses={finalStatuses}
          progressStatuses={progressStatuses}
          watch={watch}
        />
      </div>
    </div>
  );
}
