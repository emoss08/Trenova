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
import { lazy } from "react";

const CustomerInformation = lazy(() => import("./cards/customer-info"));
const ShipmentInformation = lazy(() => import("./cards/shipment-info"));
const RateCalcInformation = lazy(() => import("./cards/rate-calc-info"));
const ChargeInformation = lazy(() => import("./cards/charge-info"));

export function BillingInfoTab() {
  return (
    <ScrollArea className="h-[80vh] p-4">
      <div className="grid grid-cols-1 gap-y-8">
        <CustomerInformation />
        <ShipmentInformation />
        <RateCalcInformation />
        <ChargeInformation />
      </div>
    </ScrollArea>
  );
}
