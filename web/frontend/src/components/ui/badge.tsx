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

import { cn } from "@/lib/utils";
import { cva, type VariantProps } from "class-variance-authority";
import * as React from "react";

const badgeVariants = cva(
  "focus:ring-ring inline-flex select-none items-center gap-x-1.5 rounded-sm border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2",
  {
    variants: {
      variant: {
        default:
          "bg-primary text-primary-foreground hover:bg-primary/80 border-transparent",
        secondary:
          "bg-secondary text-secondary-foreground hover:bg-secondary/80 border-transparent",
        active:
          "border border-green-200 bg-green-200 text-green-600 dark:border-green-500 dark:bg-green-600/30 dark:text-green-400 forced-colors:outline",
        inactive:
          "border border-red-200 bg-red-200 text-red-600 dark:border-red-500 dark:bg-red-600/30 dark:text-red-400 forced-colors:outline",
        info: "border border-blue-200 bg-blue-200 text-blue-600 dark:border-blue-500 dark:bg-blue-600/30 dark:text-blue-400 forced-colors:outline",
        purple:
          "border border-purple-200 bg-purple-200 text-purple-600 dark:border-purple-500 dark:bg-purple-600/30 dark:text-purple-400 forced-colors:outline",
        pink: "border border-pink-200 bg-pink-200 text-pink-600 dark:border-pink-500 dark:bg-pink-600/30 dark:text-pink-400 forced-colors:outline",
        warning:
          "border border-yellow-200 bg-yellow-200 text-yellow-600 dark:border-yellow-500 dark:bg-yellow-600/30 dark:text-yellow-400 forced-colors:outline",
        outline: "text-foreground",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {
  withDot?: boolean;
}

function Badge({ className, variant, withDot = true, ...props }: BadgeProps) {
  return (
    <div className={cn(badgeVariants({ variant }), className)} {...props}>
      {withDot && (
        <svg
          className="size-1.5 fill-current"
          viewBox="0 0 6 6"
          aria-hidden="true"
        >
          <circle cx={3} cy={3} r={3} />
        </svg>
      )}
      {props.children}
    </div>
  );
}

export { Badge, badgeVariants };

