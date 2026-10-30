import { Button as ButtonPrimitive } from "@base-ui/react/button";
import { cn } from "@trenova/shared/lib/utils";
import { buttonVariants } from "@trenova/shared/lib/variants/button";
import { type VariantProps } from "class-variance-authority";
import { Kbd, kbdOnSolidClassName } from "./kbd";
import { Spinner } from "./spinner";

export type ButtonProps = ButtonPrimitive.Props &
  VariantProps<typeof buttonVariants> & {
    isLoading?: boolean;
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
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    >
      {isLoading ? (
        <>
          <Spinner className="size-4" />
          {loadingText}
        </>
      ) : (
        <>
          {children}
          {shortcut && (
            <Kbd className={cn("ml-0.5", SOLID_VARIANTS.has(variant) && kbdOnSolidClassName)}>
              {shortcut}
            </Kbd>
          )}
        </>
      )}
    </ButtonPrimitive>
  );
}

export { Button };
