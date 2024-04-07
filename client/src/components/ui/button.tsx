/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { cn } from "@/lib/utils";
import { IconProp } from "@fortawesome/fontawesome-svg-core";
import { faSpinner } from "@fortawesome/pro-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { Slot, Slottable } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import * as React from "react";

const buttonVariants = cva(
  "ring-offset-background focus-visible:ring-ring inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground hover:bg-primary/90",
        destructive:
          "bg-destructive text-destructive-foreground hover:bg-destructive/90",
        outline:
          "border-input bg-background hover:bg-accent hover:text-accent-foreground border",
        secondary:
          "bg-secondary text-secondary-foreground hover:bg-secondary/80",
        ghost: "hover:bg-accent hover:text-accent-foreground",
        link: "text-primary underline-offset-4 hover:underline",
        expandIcon:
          "bg-primary text-background hover:bg-primary/90 group relative",
        ringHover:
          "bg-primary text-primary-foreground hover:bg-primary/90 hover:ring-primary/90 transition-all duration-300 hover:ring-2 hover:ring-offset-2",
        shine:
          "animate-shine from-primary via-primary/75 to-primary text-primary-foreground bg-gradient-to-r bg-[length:400%_100%] ",
        gooeyRight:
          "bg-primary text-primary-foreground relative z-0 overflow-hidden from-zinc-400 transition-all duration-500 before:absolute before:inset-0 before:-z-10 before:translate-x-[150%] before:translate-y-[150%] before:scale-[2.5] before:rounded-[100%] before:bg-gradient-to-r before:transition-transform before:duration-1000  hover:before:translate-x-0 hover:before:translate-y-0 ",
        gooeyLeft:
          "bg-primary text-primary-foreground relative z-0 overflow-hidden from-zinc-400 transition-all duration-500 after:absolute after:inset-0 after:-z-10 after:translate-x-[-150%] after:translate-y-[150%] after:scale-[2.5] after:rounded-[100%] after:bg-gradient-to-l after:transition-transform after:duration-1000  hover:after:translate-x-0 hover:after:translate-y-0 ",
        linkHover1:
          "after:bg-primary relative after:absolute after:bottom-2 after:h-px after:w-2/3 after:origin-bottom-left after:scale-x-100 after:transition-transform after:duration-300 after:ease-in-out hover:after:origin-bottom-right hover:after:scale-x-0",
        linkHover2:
          "after:bg-primary relative after:absolute after:bottom-2 after:h-px after:w-2/3 after:origin-bottom-right after:scale-x-0 after:transition-transform after:duration-300 after:ease-in-out hover:after:origin-bottom-left hover:after:scale-x-100",
      },
      size: {
        default: "h-10 px-4 py-2",
        xs: "h-6 px-2",
        sm: "h-9 rounded-md px-3",
        lg: "h-11 rounded-md px-8",
        nosize: "",
        icon: "size-10",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);

interface IconProps {
  icon: IconProp;
  iconPlacement: "left" | "right";
}

interface IconRefProps {
  icon?: never;
  iconPlacement?: undefined;
}

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

export type ButtonIconProps = IconProps | IconRefProps;

const Button = React.forwardRef<
  HTMLButtonElement,
  ButtonProps & ButtonIconProps & { isLoading?: boolean; loadingText?: string }
>(
  (
    {
      className,
      variant,
      size,
      asChild = false,
      isLoading,
      loadingText,
      icon,
      iconPlacement,
      ...props
    },
    ref,
  ) => {
    const Comp = asChild ? Slot : "button";
    return (
      <Comp
        className={cn(
          buttonVariants({ variant, size, className }),
          isLoading && "cursor-progress opacity-80",
          "disabled:cursor-not-allowed",
        )}
        ref={ref}
        {...props}
      >
        {isLoading ? (
          <>
            <FontAwesomeIcon icon={faSpinner} className="mr-2 size-3" spin />
            <Slottable>{loadingText || "Saving Changes..."}</Slottable>
          </>
        ) : (
          <>
            {icon && iconPlacement === "left" && (
              <div className="group-hover:translate-x-100 w-0 translate-x-0 pr-0 opacity-0 transition-all duration-200 group-hover:w-5 group-hover:pr-2 group-hover:opacity-100">
                <FontAwesomeIcon icon={icon} className="mr-2 size-3" />
              </div>
            )}
            <Slottable>{props.children}</Slottable>
            {icon && iconPlacement === "right" && (
              <div className="w-0 translate-x-full pl-0 opacity-0 transition-all duration-200 group-hover:w-5 group-hover:translate-x-0 group-hover:pl-2 group-hover:opacity-100">
                <FontAwesomeIcon icon={icon} className="mr-2 size-3" />
              </div>
            )}
          </>
        )}
      </Comp>
    );
  },
);

Button.displayName = "Button";

export { Button, buttonVariants };
