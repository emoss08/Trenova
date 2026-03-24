import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { cn } from "@/lib/utils";
import { ExternalLinkIcon } from "lucide-react";
import React, { useState } from "react";
import { Link, type LinkProps } from "react-router";

// Small Wrapper around react router <Link> to keep up with the design system
const internalLinkStyle = {
  fontWeight: "normal",
  width: "fit-content",
  display: "inline-block",
};

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

    onClose();
  };

  return (
    <AlertDialog open={open} onOpenChange={onClose}>
      <AlertDialogContent>
        <AlertDialogTitle>External Link</AlertDialogTitle>
        <AlertDialogDescription>
          You are about to leave Trenova and visit an external website. Are you sure you want to
          continue?
        </AlertDialogDescription>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={onClose}>Cancel</AlertDialogCancel>
          <AlertDialogAction variant="destructive" onClick={() => onClick(link)}>
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
        <ExternalLinkIcon className="size-2" />
      </div>
      <ExternalLinkDialog open={open} onClose={() => setOpen(false)} link={href} />
    </>
  );
}

type EntityRedirectLinkProps = Omit<LinkProps, "to"> & {
  entityId?: string;
  baseUrl: string;
  panelOpen?: boolean;
} & React.HTMLAttributes<HTMLDivElement>;

export function EntityRedirectLink({
  entityId,
  baseUrl,
  panelOpen,
  children,
  className,
  ...rest
}: EntityRedirectLinkProps) {
  const url = React.useMemo(() => {
    let computedUrl = `${baseUrl}`;

    if (panelOpen) {
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
      preservedParams.set("panelEntityId", entityId || "");
      preservedParams.set("panelType", "edit");

      computedUrl += `?${preservedParams.toString()}`;
    } else {
      computedUrl += `/${entityId}`;
    }

    return computedUrl;
  }, [baseUrl, entityId, panelOpen]);

  const linkClassName = React.useMemo(
    () =>
      cn("inline-flex w-full items-center text-primary underline hover:text-primary/70", className),
    [className],
  );

  if (!entityId) {
    return <>{children}</>;
  }

  return (
    <Link
      to={url}
      target={panelOpen ? "_blank" : undefined}
      className={linkClassName}
      title={`View ${entityId}`}
      aria-label={`View ${entityId}`}
      style={internalLinkStyle}
      {...rest}
    >
      {children}
    </Link>
  );
}
