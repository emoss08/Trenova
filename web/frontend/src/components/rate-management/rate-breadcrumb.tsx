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



import { Button } from "@/components/ui/button";
import { useUserPermissions } from "@/context/user-permissions";
import { upperFirst } from "@/lib/utils";
import { useRateStore } from "@/stores/RateStore";
import { EllipsisVerticalIcon } from "lucide-react";
import { useCallback, useEffect } from "react";
import { FavoriteIcon } from "../layout/user-favorite";
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
  const [currentView, setCurrentView] = useRateStore.use("currentView");
  const toggleView = useCallback(() => {
    if (currentView === "list") {
      setCurrentView("map");
    } else {
      setCurrentView("list");
    }
  }, [currentView, setCurrentView]);

  // Use Effect to switch the view based on keypress
  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "s" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        toggleView();
      }
    };
    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, [toggleView]);

  useEffect(() => {
    // set the document title based on the current view
    document.title = `Rate Management - ${upperFirst(currentView)} View`;
  });

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" className="h-9 font-semibold lg:flex">
          <EllipsisVerticalIcon className="mr-1 mt-0.5 size-4" />
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

export function RateBreadcrumb() {
  const { userHasPermission } = useUserPermissions();
  //   const [open, setOpen] = useShipmentStore.use("addShipmentDialogOpen");

  return (
    <div className="flex justify-between pb-4 pt-5 md:py-4">
      <div>
        <h2 className="mt-10 flex scroll-m-20 items-center pb-2 text-xl font-semibold tracking-tight transition-colors first:mt-0">
          Rate Management
          <FavoriteIcon />
        </h2>
        <div className="flex items-center">
          <a className="text-muted-foreground hover:text-muted-foreground/80 text-sm font-medium">
            Dispatch - Rate Management
          </a>
        </div>
      </div>
      <div className="mt-3 flex">
        <OptionsDropdown />
        {userHasPermission("rate.add") ? (
          <Button
            size="sm"
            className="ml-3 h-9 font-semibold"
            onClick={() => {}}
          >
            Add New Rate
          </Button>
        ) : null}
        {/* <ShipmentSheet open={open} onOpenChange={setOpen} /> */}
      </div>
    </div>
  );
}
