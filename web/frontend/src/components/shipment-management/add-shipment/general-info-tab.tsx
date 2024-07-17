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

import { ScrollArea } from "@/components/ui/scroll-area";
import { useNextProNumber, useShipmentControl } from "@/hooks/useQueries";
import { DispatchInformationCard } from "./cards/dispatch-info";
import EquipmentInformationCard from "./cards/equipment-info";
import { GeneralInformationCard } from "./cards/general-info";
import { LocationInformationCard } from "./cards/location-info";

export function GeneralInfoTab() {
  const { data, isLoading: isShipmentControlLoading } = useShipmentControl();
  const { proNumber, isProNumberLoading } = useNextProNumber();

  return (
    <ScrollArea className="h-[80vh] p-4">
      <div className="grid grid-cols-1 gap-y-8">
        <GeneralInformationCard
          proNumber={proNumber as string}
          isProNumberLoading={isProNumberLoading}
          shipmentControlData={data}
          isShipmentControlLoading={isShipmentControlLoading}
        />
        <LocationInformationCard
          shipmentControlData={data}
          isShipmentControlLoading={isShipmentControlLoading}
        />
        <EquipmentInformationCard />
        <DispatchInformationCard
          shipmentControlData={data}
          isShipmentControlLoading={isShipmentControlLoading}
        />
      </div>
    </ScrollArea>
  );
}
