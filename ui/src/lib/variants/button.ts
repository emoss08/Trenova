import { cva } from "class-variance-authority";

export const buttonVariants = cva(
  "inline-flex items-center justify-center cursor-pointer gap-1.5 whitespace-nowrap rounded-md text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:opacity-50 disabled:cursor-not-allowed [&_svg]:pointer-events-none [&_svg]:size-3 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground hover:bg-primary/90",
        destructive:
          "bg-destructive text-destructive-foreground hover:bg-destructive/90",
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
