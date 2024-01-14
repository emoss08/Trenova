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

import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import React from "react";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faGrid2 } from "@fortawesome/pro-duotone-svg-icons";

function AppGridButton({ open }: { open: boolean }) {
  return (
    <TooltipProvider delayDuration={100}>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            size="icon"
            variant="outline"
            role="button"
            aria-label="Open Application Grid"
            aria-expanded={open}
            className="relative h-9 border border-muted-foreground/40 hover:border-muted-foreground/80"
          >
            <FontAwesomeIcon icon={faGrid2} className="h-5 w-5" />
          </Button>
        </TooltipTrigger>
        <TooltipContent side="bottom" sideOffset={5}>
          <span>Application Grid</span>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

export function AppGridMenu() {
  const [open, setOpen] = React.useState(false);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger>
        <AppGridButton open={open} />
      </PopoverTrigger>
      <PopoverContent
        className="w-80"
        sideOffset={10}
        alignOffset={-40}
        align="end"
      >
        Application Grid will be here
      </PopoverContent>
    </Popover>
  );
}
