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
import { upperFirst } from "@/lib/utils";
import { useShipmentStore } from "@/stores/ShipmentStore";
import { MoreVerticalIcon } from "lucide-react";
import React from "react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from "../ui/dropdown-menu";

function OptionsDropdown() {
  const [currentView, setCurrentView] = useShipmentStore.use("currentView");
  const toggleView = () => {
    if (currentView === "list") {
      setCurrentView("map");
    } else {
      setCurrentView("list");
    }
  };

  // Use Effect to switch the view based on keypress
  React.useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "s" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        toggleView();
      }
    };
    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, [toggleView]);

  React.useEffect(() => {
    // set the document title based on the current view
    document.title = `Shipment Management - ${upperFirst(currentView)} View`;
  });

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" className="h-9 font-semibold lg:flex">
          <MoreVerticalIcon className="mr-1 mt-0.5 h-4 w-4" />
          Options
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="w-[150px]">
        <DropdownMenuLabel>Options</DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => toggleView()}>
          Switch View
          <DropdownMenuShortcut>⌘S</DropdownMenuShortcut>
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem>Import</DropdownMenuItem>
        <DropdownMenuItem>Export</DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem>View Audit Log</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function ShipmentBreadcrumb() {
  return (
    <div className="flex justify-between pb-4 pt-5 md:py-4">
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
      <div className="mt-3 flex">
        <OptionsDropdown />
        <Button size="sm" className="ml-3 h-9 font-semibold">
          Add New Shipment
        </Button>
      </div>
    </div>
  );
}
