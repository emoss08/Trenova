import { brandMarkFor } from "@trenova/shared/components/ui/logos/registry";
import { brandfetchLogoUrl } from "@trenova/shared/lib/brandfetch";
import { cn, getNameInitials } from "@trenova/shared/lib/utils";
import { useState } from "react";

type BrandLogoProps = {
  /** Vendor web domain, e.g. "openai.com". */
  domain?: string | null;
  /** Provider preset key, which resolves a bundled mark without a domain lookup. */
  presetKey?: string | null;
  /** Used for the monogram fallback and the accessible name. */
  name: string;
  /** Rendered size in CSS pixels; a remote image is requested at 2x for sharp tiles. */
  size?: number;
  className?: string;
};

/**
 * A provider's mark, resolved in three steps: a mark we ship, then the Brandfetch
 * CDN when a client id is configured, then initials.
 *
 * The bundled step comes first on purpose. It is the only one that works offline,
 * costs no request and cannot shift the layout, so the common vendors always look
 * right even in an install that never sets VITE_BRANDFETCH_CLIENT_ID.
 */
export function BrandLogo({ domain, presetKey, name, size = 32, className }: BrandLogoProps) {
  const [failedDomain, setFailedDomain] = useState<string | null>(null);

  const Mark = brandMarkFor({ presetKey, domain });
  if (Mark) {
    // The mark stands on its own. A ringed tile behind it turns a list of
    // vendors into a row of identical boxes, which is the opposite of what a
    // logo is for.
    return (
      <span
        aria-hidden
        className={cn("text-foreground flex shrink-0 items-center justify-center", className)}
        style={{ width: size, height: size }}
      >
        <Mark className="size-full" />
      </span>
    );
  }

  const src =
    domain && failedDomain !== domain ? brandfetchLogoUrl(domain, { width: size * 2 }) : null;

  if (!src) {
    return <BrandMonogram name={name} size={size} className={className} />;
  }

  return (
    <img
      src={src}
      alt=""
      role="presentation"
      width={size}
      height={size}
      loading="lazy"
      decoding="async"
      onError={() => setFailedDomain(domain ?? null)}
      className={cn("shrink-0 object-contain", className)}
      style={{ width: size, height: size }}
    />
  );
}

export function BrandMonogram({
  name,
  size = 32,
  className,
}: Pick<BrandLogoProps, "name" | "size" | "className">) {
  return (
    <span
      aria-hidden
      className={cn(
        "text-muted-foreground flex shrink-0 items-center justify-center font-semibold tracking-tight select-none",
        className,
      )}
      style={{ width: size, height: size, fontSize: Math.max(10, Math.round(size * 0.44)) }}
    >
      {getNameInitials(name, "AI", { maxLength: 2 })}
    </span>
  );
}
