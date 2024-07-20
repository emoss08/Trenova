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

import { useHeaderStore } from "@/stores/HeaderStore";
import {
  MagnifyingGlassIcon
} from "@radix-ui/react-icons";
import { Button } from "../ui/button";
import { KeyCombo, Keys, ShortcutsProvider } from "../ui/keyboard";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "../ui/tooltip";


export function SearchButton() {
  return (
    <TooltipProvider delayDuration={100}>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            size="icon"
            variant="outline"
            aria-label="Open site search"
            aria-expanded={useHeaderStore.get("searchDialogOpen")}
            onClick={() => useHeaderStore.set("searchDialogOpen", true)}
            className="group relative flex size-8 border-muted-foreground/40 hover:border-muted-foreground/80"
          >
            <MagnifyingGlassIcon className="size-5 text-muted-foreground group-hover:text-foreground" />
          </Button>
        </TooltipTrigger>
        <TooltipContent side="bottom" sideOffset={5}>
          <span>Site Search</span>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}


export function SiteSearchInput() {
  return (
    <TooltipProvider delayDuration={100}>
      <Tooltip>
        <TooltipTrigger asChild>
          <span
            aria-label="Open site search"
            aria-expanded={useHeaderStore.get('searchDialogOpen')}
            onClick={() => useHeaderStore.set('searchDialogOpen', true)}
            className="group mt-10 hidden h-9 w-[250px] items-center justify-between rounded-md border border-muted-foreground/20 px-3 py-2 text-sm hover:cursor-pointer hover:border-muted-foreground/80 hover:bg-accent xl:flex"
          >
            <div className="flex items-center">
              <MagnifyingGlassIcon className="mr-2 size-5 text-muted-foreground group-hover:text-foreground" />
              <span className="text-muted-foreground">Search...</span>
            </div>
            <div className="pointer-events-none inline-flex select-none">
              <ShortcutsProvider os="mac">
                <KeyCombo keyNames={[Keys.Command, 'K']} />
              </ShortcutsProvider>
            </div>
          </span>
        </TooltipTrigger>
        <TooltipContent side="right" sideOffset={15}>
          <span>Site Search</span>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}