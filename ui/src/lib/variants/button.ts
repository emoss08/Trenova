import { cva } from "class-variance-authority";

export const buttonVariants = cva(
  "inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md text-xs font-medium whitespace-nowrap transition-colors focus-visible:ring-1 focus-visible:ring-ring focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-3 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground hover:bg-primary/90",
        background: "bg-background text-foreground hover:bg-background/90",
        green:
          "border border-green-500/60 bg-green-700 text-white hover:border-green-400 hover:bg-green-600 [&_svg]:text-white",
        red: "border border-red-500/60 bg-red-700 text-white hover:border-red-400 hover:bg-red-600 [&_svg]:text-white",

        destructive: "bg-destructive text-white hover:bg-destructive/90",
        outline:
          "border border-input bg-background hover:bg-accent hover:text-accent-foreground",
        secondary:
          "bg-secondary text-secondary-foreground hover:bg-secondary/80",
        ghost: "hover:bg-accent hover:text-accent-foreground",
        ghostInvert:
          "bg-accent hover:bg-muted-foreground/40 hover:text-accent-foreground",
        link: "text-primary underline-offset-4 hover:underline",
      },
      size: {
        default: "h-7 px-3 py-2",
        xs: "h-5 rounded-sm px-2 py-1 text-2xs [&_svg]:size-2.5",
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
