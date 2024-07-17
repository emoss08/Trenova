/**
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
