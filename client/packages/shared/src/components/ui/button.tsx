import { Button as ButtonPrimitive } from "@base-ui/react/button";
import { cn } from "@trenova/shared/lib/utils";
import { buttonVariants } from "@trenova/shared/lib/variants/button";
import { type VariantProps } from "class-variance-authority";
import { Kbd, kbdOnSolidClassName } from "./kbd";
import { Spinner } from "./spinner";

export type ButtonProps = ButtonPrimitive.Props &
  VariantProps<typeof buttonVariants> & {
    /**
     * The request this button started is in flight. The button is disabled and shows a
     * spinner in place of its content, in the same box: it never changes width or height.
     */
    isLoading?: boolean;
    /** What a screen reader is told while loading, such as "Saving...". It is not drawn. */
    loadingText?: string;
    shortcut?: string;
  };

const SOLID_VARIANTS = new Set<ButtonProps["variant"]>(["default", "destructive"]);

function Button({
  className,
  variant = "default",
  size = "default",
  isLoading = false,
  loadingText,
  disabled,
  shortcut,
  children,
  ...props
}: ButtonProps) {
  return (
    <ButtonPrimitive
      data-slot="button"
      disabled={disabled || isLoading}
      aria-busy={isLoading}
      data-loading={isLoading || undefined}
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
      // The content stays in the button while it works, so without this the button would be
      // named for both its label and what it is doing. While it works it is named for that.
      aria-label={isLoading && loadingText ? loadingText : props["aria-label"]}
    >
      {children}
      {shortcut && (
        <Kbd className={cn("ml-0.5", SOLID_VARIANTS.has(variant) && kbdOnSolidClassName)}>
          {shortcut}
        </Kbd>
      )}
      {isLoading ? (
        <span data-slot="button-spinner" className="absolute inset-0 grid place-items-center">
          <Spinner className="size-4 max-h-full max-w-full" />
          {loadingText ? <span className="sr-only">{loadingText}</span> : null}
        </span>
      ) : null}
    </ButtonPrimitive>
  );
}

export { Button };
