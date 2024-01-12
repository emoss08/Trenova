/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
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
import { MinusIcon, PlusIcon } from "lucide-react";
import { GoogleMap } from "@google";

// Components that control the zoom in and out for the map.
export function ShipmentMapZoom({ map }: { map: GoogleMap }) {
  if (!map) return null; // This will handle the case when map is not yet loaded

  return (
    <div className="flex flex-col space-y-2">
      <Button
        className="bg-background text-foreground hover:text-background"
        size="icon"
        onClick={() => map.setZoom(map.getZoom() + 1)}
      >
        <PlusIcon size={24} />
      </Button>
      <Button
        className="bg-background text-foreground hover:text-background"
        size="icon"
        onClick={() => map.setZoom(map.getZoom() - 1)}
      >
        <MinusIcon size={24} />
      </Button>
    </div>
  );
}
