import { LinkExternal01Icon } from "@trenova/shared/components/icons";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
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
  const t = useT();

  const onClick = (link: string) => {
    // Navigate the user to the external link
    window.open(link, "_blank", "noopener,noreferrer");

    onClose();
  };

  return (
    <AlertDialog open={open} onOpenChange={onClose}>
      <AlertDialogContent>
        <AlertDialogTitle>{t("External link")}</AlertDialogTitle>
        <AlertDialogDescription>
          {t(
            "You are about to leave Trenova and visit an external website. Are you sure you want to continue?",
          )}
        </AlertDialogDescription>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={onClose}>{t("Cancel")}</AlertDialogCancel>
          <AlertDialogAction variant="destructive" onClick={() => onClick(link)}>
            {t("Continue")}
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
} & Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, "type" | "onClick">) {
  const [open, setOpen] = useState(false);
  // A button, not a div: it sits inside sentences (a dialog's description is a paragraph) and
  // has to be reached and pressed from the keyboard.
  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className={cn(
          "ui-focus-ring inline-flex cursor-pointer items-baseline gap-x-0.5 rounded-sm text-left font-semibold text-info-foreground hover:underline",
          className,
        )}
        {...props}
      >
        {children}
        <LinkExternal01Icon aria-hidden className="size-2 self-start" />
      </button>
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
  const t = useT();
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
      cn("text-primary hover:text-primary/70 inline-flex w-full items-center underline", className),
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
      title={t("View {0}", entityId)}
      aria-label={t("View {0}", entityId)}
      style={internalLinkStyle}
      {...rest}
    >
      {children}
    </Link>
  );
}
