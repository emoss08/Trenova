/**
 * Trenova - (c) 2024 Eric Moss
 * Licensed under the Business Source License 1.1 (BSL 1.1)
 *
 * You may use this software for non-production purposes only.
 * For full license text, see the LICENSE file in the project root.
 *
 * This software will be licensed under GPLv2 or later on 2026-11-16.
 * For alternative licensing options, email: eric@trenova.app
 */

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
        className="inline-flex cursor-pointer gap-x-0.5 font-semibold text-blue-500 hover:underline"
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

// Small Wrapper around react router <Link> to keep up with the design system
export const InternalLink = React.forwardRef<HTMLAnchorElement, LinkProps>(
  (props, ref) => {
    const { children, className } = props;

    return (
      <Link
        ref={ref}
        className={cn(
          "inline-flex items-center text-primary underline",
          className,
        )}
        style={{
          textDecoration: "underline",
        }}
        {...props}
      >
        {children}
      </Link>
    );
  },
);

InternalLink.displayName = "InternalLink";
