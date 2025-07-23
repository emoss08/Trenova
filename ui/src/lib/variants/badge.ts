/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { cva } from "class-variance-authority";

export const badgeVariants = cva(
  "inline-flex select-none items-center border gap-x-1.5 px-2.5 py-0.5 rounded-md text-xs font-normal transition-colors focus:outline-hidden focus:ring-2 focus:ring-ring focus:ring-offset-2",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground hover:bg-primary/80",
        secondary:
          "bg-secondary text-secondary-foreground hover:bg-secondary/80",
        active:
          "text-green-700 bg-green-600/20 border-green-600/30 dark:text-green-400",
        inactive:
          "text-red-700 bg-red-600/20 border-red-600/30 dark:text-red-400",
        info: "text-blue-700 bg-blue-600/20 border-blue-600/30 dark:text-blue-400",
        purple:
          "text-purple-700 bg-purple-600/20 border-purple-600/30 dark:text-purple-400",
        orange:
          "text-orange-700 bg-orange-600/20 border-orange-600/30 dark:text-orange-400",
        indigo:
          "text-indigo-700 bg-indigo-600/20 border-indigo-600/30 dark:text-indigo-400",
        pink: "text-pink-700 bg-pink-600/20 border-pink-600/30 dark:text-pink-400",
        teal: "text-teal-700 bg-teal-600/20 border-teal-600/30 dark:text-teal-400",
        warning:
          "text-yellow-700 bg-yellow-600/20 border-yellow-600/30 dark:text-yellow-400",
        outline: "text-muted-foreground",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);
