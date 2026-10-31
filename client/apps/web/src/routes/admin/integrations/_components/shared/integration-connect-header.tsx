import trenovaLogo from "@/assets/logo.webp";
import { PlugIcon } from "@trenova/shared/components/icons";
import { DialogDescription, DialogHeader, DialogTitle } from "@trenova/shared/components/ui/dialog";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

/**
 * A vendor's mark for each background. `light` is drawn on a light background and `dark` on a
 * dark one; either stands in for the other when only one exists.
 */
export type IntegrationLogo = {
  light?: string;
  dark?: string;
};

type IntegrationConnectHeaderProps = {
  /** The vendor's name, drawn as initials when it has no logo. */
  name: string;
  logo: IntegrationLogo;
  title: ReactNode;
  description: ReactNode;
};

/**
 * The head of an integration's connect dialog: Trenova's mark plugged into the vendor's, then
 * the dialog's title and description. The marks are decoration; the title names the vendor.
 */
export function IntegrationConnectHeader({
  name,
  logo,
  title,
  description,
}: IntegrationConnectHeaderProps) {
  return (
    <DialogHeader className="items-center gap-4 pt-2 text-center">
      <div aria-hidden="true" className="flex items-center justify-center gap-1.5">
        <LogoTile className="w-10">
          <img src={trenovaLogo} alt="" className="size-full object-contain" />
        </LogoTile>
        <span className="text-muted-foreground flex items-center gap-1">
          <span className="bg-border h-px w-3" />
          <PlugIcon className="size-3.5" />
          <span className="bg-border h-px w-3" />
        </span>
        <IntegrationLogoMark name={name} logo={logo} />
      </div>
      <div className="flex flex-col items-center gap-1.5">
        <DialogTitle>{title}</DialogTitle>
        <DialogDescription className="text-balance">{description}</DialogDescription>
      </div>
    </DialogHeader>
  );
}

function LogoTile({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <span
      data-slot="integration-logo"
      className={cn(
        "border-border bg-background flex h-10 min-w-10 shrink-0 items-center justify-center rounded-lg border p-1.5",
        className,
      )}
    >
      {children}
    </span>
  );
}

/** A vendor's logo in a bordered tile, or its initials when it has none. */
export function IntegrationLogoMark({ name, logo }: { name: string; logo: IntegrationLogo }) {
  if (!logo.light && !logo.dark) {
    return (
      <LogoTile className="text-muted-foreground text-xs font-semibold">
        {name.slice(0, 2).toUpperCase()}
      </LogoTile>
    );
  }

  // A wordmark widens its tile up to a limit; a square mark keeps the tile square.
  return (
    <LogoTile className="max-w-36">
      <ThemedLogo logo={logo} alt="" className="h-full w-auto max-w-full object-contain" />
    </LogoTile>
  );
}

/**
 * Draws the logo variant for the theme on screen. Both variants are in the page and the
 * `.dark` class picks one, so the system theme is followed too, which a check of the
 * theme setting (`theme === "dark"`) misses. Draws nothing when the vendor has no logo.
 */
export function ThemedLogo({
  logo,
  alt,
  className,
  loading,
}: {
  logo: IntegrationLogo;
  alt: string;
  className?: string;
  loading?: "lazy" | "eager";
}) {
  const light = logo.light ?? logo.dark;
  const dark = logo.dark ?? logo.light;

  if (!light || !dark) {
    return null;
  }

  if (light === dark) {
    return <img src={light} alt={alt} loading={loading} className={className} />;
  }

  return (
    <>
      <img src={light} alt={alt} loading={loading} className={cn(className, "dark:hidden")} />
      <img src={dark} alt={alt} loading={loading} className={cn(className, "hidden dark:block")} />
    </>
  );
}
