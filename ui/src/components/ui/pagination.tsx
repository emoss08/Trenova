/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import * as React from "react";

import { ButtonProps } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { buttonVariants } from "@/lib/variants/button";
import {
  faChevronLeft,
  faChevronRight,
} from "@fortawesome/pro-regular-svg-icons";
import { DotsHorizontalIcon } from "@radix-ui/react-icons";
import { VariantProps } from "class-variance-authority";
import { Icon } from "./icons";

function Pagination({ className, ...props }: React.ComponentProps<"nav">) {
  return (
    <nav
      role="navigation"
      aria-label="pagination"
      data-slot="pagination"
      className={cn(
        "mx-auto flex w-full justify-center flex-row items-center gap-1 aria-disabled:opacity-50",
        className,
      )}
      {...props}
    />
  );
}

function PaginationContent({
  className,
  ...props
}: React.ComponentProps<"ul">) {
  return (
    <ul
      data-slot="pagination-content"
      className={cn("flex flex-row items-center gap-1 select-none", className)}
      {...props}
    />
  );
}

function PaginationItem({ className, ...props }: React.ComponentProps<"li">) {
  return (
    <li
      data-slot="pagination-item"
      className={cn("hover:cursor-pointer select-none", className)}
      {...props}
    />
  );
}

type PaginationLinkProps = {
  isActive?: boolean;
  disabled?: boolean;
  variant?: VariantProps<typeof buttonVariants>["variant"];
} & Pick<ButtonProps, "size"> &
  React.ComponentProps<"a">;

function PaginationLink({
  className,
  isActive,
  size = "icon",
  disabled = false,
  variant,
  ...props
}: PaginationLinkProps) {
  return (
    <PaginationItem>
      <a
        aria-current={isActive ? "page" : undefined}
        role="link"
        aria-disabled={disabled}
        className={cn(
          buttonVariants({
            variant: variant ?? (isActive ? "default" : "ghost"),
            size,
          }),
          disabled &&
            "cursor-not-allowed pointer-events-none select-none opacity-50 bg-muted",
          className,
        )}
        {...props}
      />
    </PaginationItem>
  );
}

function PaginationPrevious({
  className,
  ...props
}: React.ComponentProps<typeof PaginationLink>) {
  return (
    <PaginationLink
      aria-label="Go to previous page"
      className={cn("gap-1 px-1.5 select-none", className)}
      {...props}
    >
      <Icon icon={faChevronLeft} className="size-4" />
    </PaginationLink>
  );
}

function PaginationNext({
  className,
  ...props
}: React.ComponentProps<typeof PaginationLink>) {
  return (
    <PaginationLink
      aria-label="Go to next page"
      className={cn("gap-1 px-1.5 select-none", className)}
      {...props}
    >
      <Icon icon={faChevronRight} className="size-4" />
    </PaginationLink>
  );
}

function PaginationEllipsis({
  className,
  ...props
}: React.ComponentProps<"span">) {
  return (
    <span
      aria-hidden
      data-slot="pagination-ellipsis"
      className={cn(
        "text-foreground px-1.5 font-semibold select-none",
        className,
      )}
      {...props}
    >
      <DotsHorizontalIcon className="inline-block size-4 align-bottom" />
      <span className="sr-only">More pages</span>
    </span>
  );
}

export {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
};
