/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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

import { Button } from "@/components/ui/button";
import { useShipmentStore } from "@/stores/ShipmentStore";

export function ShipmentBreadcrumb() {
  const [currentView, setCurrentView] = useShipmentStore.use("currentView");

  const toggleView = () => {
    if (currentView === "list") {
      setCurrentView("map");
    } else {
      setCurrentView("list");
    }
  };

  return (
    <div className="flex justify-between pt-5 pb-4 md:pt-4 md:pb-4">
      <div>
        <h2 className="mt-10 scroll-m-20 pb-2 text-xl font-semibold tracking-tight transition-colors first:mt-0">
          Shipment Management
        </h2>
        <div className="flex items-center">
          <a className="text-sm font-medium text-muted-foreground hover:text-muted-foreground/80">
            Shipment Management - Shipment Management
          </a>
        </div>
      </div>
      <div className="mt-3">
        <Button
          size="sm"
          variant="outline"
          className="h-9 font-semibold"
          onClick={toggleView}
        >
          Change View
        </Button>
        <Button size="sm" className="h-9 ml-3 font-semibold">
          Add New Shipment
        </Button>
      </div>
    </div>
  );
}
