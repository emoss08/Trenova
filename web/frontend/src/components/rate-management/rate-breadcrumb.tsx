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
