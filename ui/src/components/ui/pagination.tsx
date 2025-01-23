import * as React from "react";

import { ButtonProps } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { buttonVariants } from "@/lib/variants/button";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  DotsHorizontalIcon,
} from "@radix-ui/react-icons";
import { VariantProps } from "class-variance-authority";

const Pagination = ({ className, ...props }: React.ComponentProps<"nav">) => (
  <nav
    role="navigation"
    aria-label="pagination"
    className={cn(
      "mx-auto flex w-full justify-center flex-row items-center gap-1 aria-disabled:opacity-50",
      className,
    )}
    {...props}
  />
);

const PaginationContent = React.forwardRef<
  HTMLUListElement,
  React.ComponentProps<"ul">
>(({ className, ...props }, ref) => (
  <ul
    ref={ref}
    className={cn("flex flex-row items-center gap-1 select-none", className)}
    {...props}
  />
));
PaginationContent.displayName = "PaginationContent";

const PaginationItem = React.forwardRef<
  HTMLLIElement,
  React.ComponentProps<"li">
>(({ className, ...props }, ref) => (
  <li
    ref={ref}
    className={cn("hover:cursor-pointer select-none", className)}
    {...props}
  />
));
PaginationItem.displayName = "PaginationItem";

type PaginationLinkProps = {
  isActive?: boolean;
  disabled?: boolean;
  variant?: VariantProps<typeof buttonVariants>["variant"];
} & Pick<ButtonProps, "size"> &
  React.ComponentProps<"a">;

const PaginationLink = ({
  className,
  isActive,
  size = "icon",
  disabled = false,
  variant,
  ...props
}: PaginationLinkProps) => (
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
PaginationLink.displayName = "PaginationLink";

const PaginationPrevious = ({
  className,
  ...props
}: React.ComponentProps<typeof PaginationLink>) => (
  <PaginationLink
    aria-label="Go to previous page"
    className={cn("gap-1 px-1.5 select-none", className)}
    {...props}
  >
    <ChevronLeftIcon className="size-4" />
  </PaginationLink>
);
PaginationPrevious.displayName = "PaginationPrevious";

const PaginationNext = ({
  className,
  ...props
}: React.ComponentProps<typeof PaginationLink>) => (
  <PaginationLink
    aria-label="Go to next page"
    className={cn("gap-1 px-1.5 select-none", className)}
    {...props}
  >
    <ChevronRightIcon className="size-4" />
  </PaginationLink>
);

const PaginationEllipsis = ({
  className,
  ...props
}: React.ComponentProps<"span">) => (
  <span
    aria-hidden
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

export {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious
};

