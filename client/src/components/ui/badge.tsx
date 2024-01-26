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
import { cn } from "@/lib/utils";
import { cva, type VariantProps } from "class-variance-authority";
import * as React from "react";

const badgeVariants = cva(
  "inline-flex items-center gap-x-1.5 rounded-sm border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2",
  {
    variants: {
      variant: {
        default:
          "border-transparent bg-primary text-primary-foreground hover:bg-primary/80",
        secondary:
          "border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/80",
        destructive:
          "border-transparent bg-destructive text-destructive-foreground hover:bg-destructive/80",
        active:
          "border border-lime-400/60 bg-lime-400/30 text-lime-700 group-data-[hover]:bg-lime-400/30 dark:bg-lime-800/10 dark:text-lime-300 dark:group-data-[hover]:bg-lime-400/15 forced-colors:outline",
        inactive:
          "border border-rose-400/60 bg-rose-400/30 text-rose-700 group-data-[hover]:bg-rose-400/25 dark:bg-rose-400/10 dark:text-rose-400 dark:group-data-[hover]:bg-rose-400/20 forced-colors:outline",
        info: "border border-blue-400/60 bg-blue-400/30 text-blue-700 group-data-[hover]:bg-blue-400/25 dark:bg-blue-800/10 dark:text-blue-400 dark:group-data-[hover]:bg-blue-400/20 forced-colors:outline",
        warning:
          "border border-yellow-400/60 bg-yellow-400/30 text-yellow-700 group-data-[hover]:bg-yellow-400/25 dark:bg-yellow-800/10 dark:text-yellow-400 dark:group-data-[hover]:bg-yellow-400/20 forced-colors:outline",
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
