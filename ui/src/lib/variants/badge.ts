import { cva } from "class-variance-authority";

export const badgeVariants = cva(
  "inline-flex select-none items-center gap-x-1.5 rounded-sm border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-hidden focus:ring-2 focus:ring-ring focus:ring-offset-2",
  {
    variants: {
      variant: {
        default:
          "border-transparent bg-primary text-primary-foreground hover:bg-primary/80",
        secondary:
          "border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/80",
        active:
          "border border-green-400 bg-green-200 text-green-700 dark:bg-green-600/30 dark:text-green-400 forced-colors:outline",
        inactive:
          "border border-red-400 bg-red-200 text-red-700 dark:bg-red-600/30 dark:text-red-400 forced-colors:outline",
        info: "border border-blue-400 bg-blue-200 text-blue-700 dark:bg-blue-600/30 dark:text-blue-400 forced-colors:outline",
        purple:
          "border border-purple-400 bg-purple-200 text-purple-700 dark:bg-purple-600/30 dark:text-purple-400 forced-colors:outline",
        pink: "border border-pink-400 bg-pink-200 text-pink-700 dark:bg-pink-600/30 dark:text-pink-400 forced-colors:outline",
        teal: "border border-teal-400 bg-teal-200 text-teal-700 dark:bg-teal-600/30 dark:text-teal-400 forced-colors:outline",
        warning:
          "border border-yellow-400 bg-yellow-200 text-yellow-700 dark:bg-yellow-600/30 dark:text-yellow-400 forced-colors:outline",
        outline: "text-muted-foreground",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);
