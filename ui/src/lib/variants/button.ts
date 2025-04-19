import { cva } from "class-variance-authority";

export const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap cursor-pointer rounded-md text-xs font-medium transition-colors focus-visible:outline-hidden focus-visible:ring-1 focus-visible:ring-blue-500 disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-3 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default:
          "border bg-blue-300 hover:bg-blue-400/70 hover:border-blue-600 border-blue-500 text-black [&_svg]:text-black dark:bg-blue-700 dark:border-blue-500/60 dark:hover:bg-blue-600 dark:hover:border-blue-400 dark:text-white dark:[&_svg]:text-white",
        destructive:
          "border border-red-500 bg-red-600 text-white hover:bg-red-500",
        outline:
          "border border-input bg-background hover:bg-accent hover:text-accent-foreground",
        secondary:
          "bg-secondary text-secondary-foreground hover:bg-secondary/80",
        ghost: "hover:bg-accent hover:text-accent-foreground",
        link: "text-primary underline-offset-4 hover:underline",
      },
      size: {
        default: "h-7 px-3 py-2",
        xs: "h-5 px-2 py-1 rounded-sm text-2xs [&_svg]:size-2.5",
        sm: "h-6 rounded-md px-3 text-2xs",
        lg: "h-8 rounded-md px-3",
        xl: "h-9 rounded-md px-3",
        noSize: "",
        icon: "size-8",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);
