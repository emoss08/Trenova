import { cva } from "class-variance-authority";

export const badgeVariants = cva(
  "inline-flex select-none items-center gap-x-1.5 px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-hidden focus:ring-2 focus:ring-ring focus:ring-offset-2 [&_svg]:size-2",
  {
    variants: {
      variant: {
        default:
          "border-transparent bg-primary text-primary-foreground hover:bg-primary/80",
        secondary:
          "border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/80",
        active: "[&_svg]:text-green-600",
        inactive: "[&_svg]:text-red-600",
        info: "[&_svg]:text-blue-600",
        purple: "[&_svg]:text-purple-600",
        pink: "[&_svg]:text-pink-600",
        teal: "[&_svg]:text-teal-600",
        warning: "[&_svg]:text-yellow-600",
        outline: "text-muted-foreground",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);
