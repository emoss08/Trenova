import { cva } from "class-variance-authority";

export const buttonVariants = cva(
  "ui-focus-ring inline-flex shrink-0 cursor-pointer items-center justify-center gap-2 rounded-md text-sm font-medium whitespace-nowrap ui-press outline-none select-none disabled:pointer-events-none disabled:opacity-50 aria-invalid:border-danger aria-invalid:bg-danger/10 aria-invalid:[--ring:var(--ring-danger)] [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        default:
          "bg-ink text-ink-foreground shadow-key hover:bg-ink-hover active:bg-ink-active",
        destructive:
          "bg-danger text-foreground-on-solid shadow-key hover:bg-danger-hover [--ring:var(--ring-danger)]",
        outline:
          "border border-input bg-card shadow-raised hover:border-border-strong hover:bg-surface-hover active:bg-surface-active",
        secondary:
          "bg-surface-active/70 text-secondary-foreground hover:bg-surface-active active:bg-border",
        ghost: "hover:bg-surface-hover hover:text-accent-foreground active:bg-surface-active",
        ghostInvert: "bg-accent hover:bg-surface-active hover:text-accent-foreground",
        link: "text-brand underline-offset-4 hover:underline active:scale-100",
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
