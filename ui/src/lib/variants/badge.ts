import { cva } from "class-variance-authority";

export const badgeVariants = cva(
  "inline-flex select-none items-center gap-x-1.5 px-2.5 py-0.5 rounded-md text-xs font-normal transition-colors focus:outline-hidden focus:ring-2 focus:ring-ring focus:ring-offset-2 [&_svg]:size-2",
  {
    variants: {
      variant: {
        default:
          "border-transparent bg-primary text-primary-foreground hover:bg-primary/80",
        secondary:
          "border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/80",
        active:
          "[&_svg]:text-green-600 dark:bg-green-600/20 dark:text-green-400",
        inactive: "[&_svg]:text-red-600 dark:bg-red-600/20 dark:text-red-400",
        info: "[&_svg]:text-blue-600 dark:bg-blue-600/20 dark:text-blue-400",
        purple:
          "[&_svg]:text-purple-600 dark:bg-purple-600/20 dark:text-purple-400",
        pink: "[&_svg]:text-pink-600 dark:bg-pink-600/20 dark:text-pink-400",
        teal: "[&_svg]:text-teal-600 dark:bg-teal-600/20 dark:text-teal-400",
        warning:
          "[&_svg]:text-yellow-600 dark:bg-yellow-600/20 dark:text-yellow-400",
        outline: "text-muted-foreground",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);
