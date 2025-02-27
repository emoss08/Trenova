import { Slot } from "@radix-ui/react-slot";
import { type VariantProps } from "class-variance-authority";
import * as React from "react";

import { cn } from "@/lib/utils";
import { buttonVariants } from "@/lib/variants/button";
import { PulsatingDots } from "./pulsating-dots";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "./tooltip";

export type ButtonProps = React.ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean;
    isLoading?: boolean;
    loadingText?: string;
  };

function Button({
  className,
  variant,
  size,
  asChild = false,
  isLoading = false,
  loadingText = "Saving Changes...",
  children,
  ...props
}: ButtonProps) {
  const Comp = asChild ? Slot : "button";
  return (
    <Comp
      data-slot="button"
      className={cn(buttonVariants({ variant, size, className }))}
      disabled={isLoading}
      {...props}
    >
      {isLoading && <PulsatingDots size={1} color="background" />}
      {isLoading && loadingText && loadingText}
      {!isLoading && children}
    </Comp>
  );
}

Button.displayName = "Button";

export { Button };

export function FormSaveButton({
  title,
  isPopout,
  isSubmitting,
}: {
  title: string;
  isPopout: boolean;
  isSubmitting: boolean;
}) {
  return (
    <TooltipProvider delayDuration={0}>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button type="submit" isLoading={isSubmitting}>
            Save {isPopout ? "and Close" : "Changes"}
          </Button>
        </TooltipTrigger>
        <TooltipContent className="flex items-center gap-2">
          <kbd className="-me-1 inline-flex h-5 max-h-full items-center rounded bg-muted-foreground/60 px-1 font-[inherit] text-[0.625rem] font-medium text-foreground">
            Ctrl
          </kbd>
          <kbd className="-me-1 inline-flex h-5 max-h-full items-center rounded bg-muted-foreground/60 px-1 font-[inherit] text-[0.625rem] font-medium text-foreground">
            Enter
          </kbd>
          <p>to save and close the {title}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
