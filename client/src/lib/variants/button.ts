import { cva } from "class-variance-authority";

export const buttonVariants = cva(
  "inline-flex shrink-0 items-center justify-center gap-2 rounded-md text-sm font-medium whitespace-nowrap transition-all outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:pointer-events-none disabled:opacity-50 aria-invalid:border-destructive aria-invalid:bg-destructive/20 aria-invalid:ring-destructive/20 aria-invalid:data-pressed:border-destructive dark:aria-invalid:ring-destructive/20 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        default: "bg-brand text-brand-foreground hover:bg-brand/90",
        destructive:
          "bg-destructive text-white hover:bg-destructive/90 focus-visible:ring-destructive/20 dark:text-white dark:focus-visible:ring-destructive/40",
        outline: "border border-input bg-background hover:bg-muted hover:text-accent-foreground",
        secondary: "bg-secondary text-secondary-foreground hover:bg-secondary/80",
        ghost: "hover:bg-accent hover:text-accent-foreground dark:hover:bg-accent/50",
        ghostInvert: "bg-accent hover:bg-muted-foreground/40 hover:text-accent-foreground",
        link: "text-primary underline-offset-4 hover:underline",
      },
      size: {
        default: "h-8 px-3.5 py-1.5 has-[>svg]:px-2.5",
        xxxs: "h-4.5 px-1.5 py-1 has-[>svg]:px-1",
        xxs: "h-5.5 px-2 py-1 has-[>svg]:px-1.5",
        xs: "h-6 px-2.5 py-1 has-[>svg]:px-2",
        sm: "h-7 gap-1.5 rounded-md px-2.5 has-[>svg]:px-2",
        lg: "h-9 rounded-md px-5 has-[>svg]:px-3.5",
        icon: "size-8",
        "icon-xxs": "size-3.5",
        "icon-xs": "size-5.5",
        "icon-sm": "size-7",
        "icon-lg": "size-9",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);
