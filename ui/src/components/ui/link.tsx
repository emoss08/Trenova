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
          "inline-flex w-full items-center text-primary hover:text-primary/70 underline",
          className,
        )}
        style={{
          fontWeight: "normal",
          width: "fit-content",
          display: "inline-block",
        }}
        {...props}
      >
        {children}
      </Link>
    );
  },
);

InternalLink.displayName = "InternalLink";

type EntityRedirectLinkProps = Omit<LinkProps, "to"> & {
  entityId?: string;
  baseUrl: string;
  modelOpen?: boolean;
};

export const EntityRedirectLink = React.forwardRef<
  HTMLAnchorElement,
  EntityRedirectLinkProps
>(({ entityId, baseUrl, modelOpen, children, className, ...rest }, ref) => {
  if (!entityId) {
    return <>{children}</>;
  }

  let url = `${baseUrl}`;

  if (modelOpen) {
    url += `?entityId=${entityId}&modal=edit`;
  } else {
    url += `/${entityId}`;
  }

  return (
    <Link
      ref={ref}
      to={url}
      className={cn(
        "inline-flex w-full items-center text-primary hover:text-primary/70 underline",
        className,
      )}
      style={{
        fontWeight: "normal",
        width: "fit-content",
        display: "inline-block",
      }}
      {...rest}
    >
      {children}
    </Link>
  );
});

EntityRedirectLink.displayName = "EntityRedirectLink";
