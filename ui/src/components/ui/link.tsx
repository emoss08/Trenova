/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { cn } from "@/lib/utils";
import { faExternalLinkAlt } from "@fortawesome/pro-regular-svg-icons";
import React, { useState } from "react";
import { Link, LinkProps } from "react-router-dom";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "./alert-dialog";
import { Icon } from "./icons";

function ExternalLinkDialog({
  open,
  onClose,
  link,
}: {
  open: boolean;
  onClose: () => void;
  link: string;
}) {
  const onClick = (link: string) => {
    // Navigate the user to the external link
    window.open(link, "_blank", "noopener,noreferrer");
  };

  return (
    <AlertDialog open={open} onOpenChange={onClose}>
      <AlertDialogContent>
        <AlertDialogTitle>External Link</AlertDialogTitle>
        <AlertDialogDescription>
          You are about to leave Trenova and visit an external website. Are you
          sure you want to continue?
        </AlertDialogDescription>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={onClose}>Cancel</AlertDialogCancel>
          <AlertDialogAction onClick={() => onClick(link)}>
            Continue
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

export function ExternalLink({
  href,
  children,
  className,
  ...props
}: {
  href: string;
  children: React.ReactNode;
} & React.HTMLAttributes<HTMLDivElement>) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <div
        onClick={() => setOpen(true)}
        className={cn(
          "inline-flex cursor-pointer gap-x-0.5 font-semibold text-blue-500 hover:underline",
          className,
        )}
        {...props}
      >
        {children}
        <Icon icon={faExternalLinkAlt} className="size-2" />
      </div>
      <ExternalLinkDialog
        open={open}
        onClose={() => setOpen(false)}
        link={href}
      />
    </>
  );
}

type InteralLinkProps = LinkProps & React.ComponentProps<"a">;

// Small Wrapper around react router <Link> to keep up with the design system
const internalLinkStyle = {
  fontWeight: "normal",
  width: "fit-content",
  display: "inline-block",
};

// Small Wrapper around react router <Link> to keep up with the design system
export const InternalLink = React.memo(function InternalLink({
  children,
  className,
  ...props
}: InteralLinkProps) {
  const linkClassName = React.useMemo(
    () =>
      cn(
        "inline-flex w-full items-center text-primary hover:text-primary/70 underline",
        className,
      ),
    [className],
  );

  return (
    <Link className={linkClassName} style={internalLinkStyle} {...props}>
      {children}
    </Link>
  );
});
type EntityRedirectLinkProps = Omit<LinkProps, "to"> & {
  entityId?: string;
  baseUrl: string;
  modelOpen?: boolean;
} & React.HTMLAttributes<HTMLDivElement>;

export const EntityRedirectLink = React.memo(function EntityRedirectLink({
  entityId,
  baseUrl,
  modelOpen,
  children,
  className,
  ...rest
}: EntityRedirectLinkProps) {
  const url = React.useMemo(() => {
    let computedUrl = `${baseUrl}`;

    if (modelOpen) {
      // Get current URL parameters to preserve them
      const currentParams = new URLSearchParams(window.location.search);
      const preservedParams = new URLSearchParams();

      // Preserve pagination parameters
      if (currentParams.has("page")) {
        preservedParams.set("page", currentParams.get("page")!);
      }
      if (currentParams.has("pageSize")) {
        preservedParams.set("pageSize", currentParams.get("pageSize")!);
      }

      // Add entity modal parameters
      preservedParams.set("entityId", entityId || "");
      preservedParams.set("modal", "edit");

      computedUrl += `?${preservedParams.toString()}`;
    } else {
      computedUrl += `/${entityId}`;
    }

    return computedUrl;
  }, [baseUrl, entityId, modelOpen]);

  const linkClassName = React.useMemo(
    () =>
      cn(
        "inline-flex w-full items-center text-primary hover:text-primary/70 underline",
        className,
      ),
    [className],
  );

  if (!entityId) {
    return <>{children}</>;
  }

  return (
    <Link
      to={url}
      target={modelOpen ? "_blank" : undefined}
      className={linkClassName}
      title={`View ${entityId}`}
      aria-label={`View ${entityId}`}
      style={internalLinkStyle}
      {...rest}
    >
      {children}
    </Link>
  );
});
