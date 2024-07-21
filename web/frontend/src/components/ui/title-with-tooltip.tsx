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

import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { faCircleInfo } from "@fortawesome/pro-duotone-svg-icons";
import { Icon } from "../common/icons";

interface TitleWithTooltipProps extends React.HTMLAttributes<HTMLDivElement> {
  title: string;
  tooltip: string;
}

export function TitleWithTooltip({
  title,
  tooltip,
  className,
}: TitleWithTooltipProps) {
  return (
    <div className={cn("flex items-center space-x-1.5")}>
      <h2 className={cn("text-lg font-semibold", className)}>{title}</h2>
      <TooltipProvider delayDuration={100}>
        <Tooltip>
          <TooltipTrigger asChild>
            <Icon
              icon={faCircleInfo}
              className="text-foreground mb-0.5 size-3.5"
            />
          </TooltipTrigger>
          <TooltipContent>
            <span>{tooltip}</span>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </div>
  );
}
