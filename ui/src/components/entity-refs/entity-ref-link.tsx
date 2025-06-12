import { cn } from "@/lib/utils";
import React, { memo } from "react";
import { Link } from "react-router";

type EntityRefLinkInnerProps = React.ComponentProps<typeof Link>;

export function EntityRefLinkInner({
  className,
  children,
  ...props
}: EntityRefLinkInnerProps) {
  return (
    <Link
      {...props}
      className={cn("max-w-px cursor-pointer text-left", className)}
    >
      {children}
    </Link>
  );
}

export function EntityRefLinkDisplayText({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <span className="text-sm font-normal text-nowrap underline hover:text-foreground/70">
      {children}
    </span>
  );
}

export function EntityRefLinkColor({
  color,
  displayText,
}: {
  color: string;
  displayText: string;
}) {
  return (
    <EntityRefLinkColorInner>
      <div
        className="size-2 rounded-full"
        style={{
          backgroundColor: color,
        }}
      />
      <p>{displayText}</p>
    </EntityRefLinkColorInner>
  );
}

const EntityRefLinkColorInner = memo(function EntityRefLinkColorInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center gap-x-1.5 text-sm font-normal text-foreground underline hover:text-foreground/70">
      {children}
    </div>
  );
});
