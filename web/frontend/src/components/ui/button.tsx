/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
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
        active:
          "border border-green-200 bg-green-200 text-green-600 dark:border-green-500 dark:bg-green-600/30 dark:text-green-400",
        warning:
          "border border-yellow-200 bg-yellow-200 text-yellow-600 dark:border-yellow-500 dark:bg-yellow-600/30 dark:text-yellow-400",
        pink: "border border-pink-200 bg-pink-200 text-pink-600 dark:border-pink-500 dark:bg-pink-600/30 dark:text-pink-400",
        inactive:
          "border border-red-200 bg-red-200 text-red-600 dark:border-red-500 dark:bg-red-600/30 dark:text-red-400",
        purple:
          "border border-purple-200 bg-purple-200 text-purple-600 dark:border-purple-500 dark:bg-purple-600/30 dark:text-purple-400",
        info: "border border-blue-200 bg-blue-200 text-blue-600 dark:border-blue-500 dark:bg-blue-600/30 dark:text-blue-400",
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

