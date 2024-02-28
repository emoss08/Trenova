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

import { ExternalLinkIcon } from "@radix-ui/react-icons";
import React from "react";
import { Link } from "react-router-dom";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";

export function ExternalLinkDialog({
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
        <AlertDialogHeader>
          <AlertDialogTitle>External Link</AlertDialogTitle>
          <AlertDialogDescription>
            You are about to leave Trenova and visit an external website. Are
            you sure you want to continue?
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
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
  const [open, setOpen] = React.useState(false);
  return (
    <>
      <div
        onClick={() => setOpen(true)}
        className="text-foreground-600 inline-flex cursor-pointer items-center gap-x-0.5 font-semibold underline decoration-blue-600"
        {...props}
      >
        {children}
        <ExternalLinkIcon />
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
export const InternalLink = React.forwardRef<
  HTMLAnchorElement,
  React.ComponentProps<typeof Link>
>((props, ref) => (
  <Link
    ref={ref}
    className="inline-flex items-center font-semibold text-blue-600 hover:underline"
    {...props}
  >
    {props.children}
  </Link>
));
