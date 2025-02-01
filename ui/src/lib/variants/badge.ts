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
        active: "text-green-600 bg-green-600/20 dark:text-green-400",
        inactive: "text-red-600 bg-red-600/20 dark:text-red-400",
        info: "text-blue-600 bg-blue-600/20 dark:text-blue-400",
        purple: "text-purple-600 bg-purple-600/20 dark:text-purple-400",
        pink: "text-pink-600 bg-pink-600/20 dark:text-pink-400",
        teal: "text-teal-600 bg-teal-600/20 dark:text-teal-400",
        warning: "text-yellow-600 bg-yellow-600/20 dark:text-yellow-400",
        outline: "text-muted-foreground",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);
