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
      {isLoading && <PulsatingDots size={1} color="foreground" />}
      {isLoading && loadingText && loadingText}
      {!isLoading && children}
    </Comp>
  );
}

Button.displayName = "Button";

type FormSaveButtonProps = {
  title: string;
  isPopout?: boolean;
  isSubmitting?: boolean;
  tooltipPosition?: "top" | "bottom" | "left" | "right";
  type?: React.ButtonHTMLAttributes<HTMLButtonElement>["type"];
  text?: string;
  onClick?: () => void;
} & VariantProps<typeof buttonVariants> &
  React.ComponentProps<"button">;

function FormSaveButton({
  title,
  isPopout = false,
  isSubmitting,
  tooltipPosition = "top",
  type = "submit",
  text,
  onClick,
  ...props
}: FormSaveButtonProps) {
  return (
    <TooltipProvider>
      <Tooltip delayDuration={500}>
        <TooltipTrigger asChild>
          <Button
            type={type}
            isLoading={isSubmitting}
            disabled={isSubmitting}
            onClick={onClick}
            {...props}
          >
            {text ? text : `Save ${isPopout ? "and Close" : "Changes"}`}
          </Button>
        </TooltipTrigger>
        <TooltipContent
          side={tooltipPosition}
          className="flex items-center gap-2 text-xs"
        >
          <kbd className="-me-1 inline-flex h-5 max-h-full items-center rounded bg-background px-1 font-[inherit] text-[0.625rem] font-medium text-foreground">
            Ctrl
          </kbd>
          <kbd className="-me-1 inline-flex h-5 max-h-full items-center rounded bg-background px-1 font-[inherit] text-[0.625rem] font-medium text-foreground">
            Enter
          </kbd>
          <p>to save and close the {title}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

export { Button, FormSaveButton };

