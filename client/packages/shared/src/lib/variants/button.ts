import { cva } from "class-variance-authority";

/**
 * What every sized button shares: centred content, the control radius, label
 * type, the press, and a 16px icon unless the icon names its own size. The
 * bare size leaves all of it out, for a control that draws its own shape.
 */
const shape =
  "justify-center gap-2 rounded-md text-sm font-medium whitespace-nowrap ui-press [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4";

export const buttonVariants = cva(
  "ui-focus-ring inline-flex shrink-0 cursor-pointer items-center outline-none select-none disabled:pointer-events-none disabled:opacity-50 aria-invalid:border-danger aria-invalid:bg-danger-subtle aria-invalid:[--ring:var(--ring-danger)]",
  {
    variants: {
      variant: {
        default:
          "bg-ink text-ink-foreground hover:bg-ink-hover active:bg-ink-active",
        destructive:
          "bg-danger text-foreground-on-solid hover:bg-danger-hover [--ring:var(--ring-danger)]",
        outline:
          "border border-input bg-card hover:border-border-strong hover:bg-surface-hover active:bg-surface-active",
        secondary:
          "bg-surface-active/70 text-secondary-foreground hover:bg-surface-active active:bg-border",
        ghost: "hover:bg-surface-hover hover:text-accent-foreground active:bg-surface-active",
        ghostInvert: "bg-accent hover:bg-surface-hover hover:text-accent-foreground",
        /** A ghost that rests quieter than the text around it and comes up to it on hover. */
        quiet:
          "text-muted-foreground hover:bg-surface-hover hover:text-foreground active:bg-surface-active",
        link: "text-brand underline-offset-4 hover:underline active:scale-100",
        /** No fill and no colour of its own: the control's classes say how it looks. */
        bare: "",
      },
      size: {
        default: `${shape} h-8 px-3.5 py-1.5 has-[>svg]:px-2.5`,
        xxxs: `${shape} h-4.5 px-1.5 py-1 has-[>svg]:px-1`,
        xxs: `${shape} h-5.5 px-2 py-1 has-[>svg]:px-1.5`,
        xs: `${shape} h-6 px-2.5 py-1 has-[>svg]:px-2`,
        sm: `${shape} h-7 gap-1.5 rounded-md px-2.5 has-[>svg]:px-2`,
        lg: `${shape} h-9 rounded-md px-5 has-[>svg]:px-3.5`,
        icon: `${shape} size-8`,
        "icon-xxs": `${shape} size-3.5`,
        "icon-xs": `${shape} size-5.5`,
        "icon-sm": `${shape} size-7`,
        "icon-lg": `${shape} size-9`,
        /** No size, padding, radius or type: for a row, tab or chip that sets its own. */
        bare: "",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);
