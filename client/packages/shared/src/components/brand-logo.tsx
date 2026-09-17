import { brandfetchLogoUrl } from "@trenova/shared/lib/brandfetch";
import { cn, getNameInitials } from "@trenova/shared/lib/utils";
import { useState } from "react";

type BrandLogoProps = {
  /** Vendor web domain, e.g. "openai.com". Without one the monogram renders. */
  domain?: string | null;
  /** Used for the monogram fallback and the accessible name. */
  name: string;
  /** Rendered size in CSS pixels; the image is requested at 2x for sharp tiles. */
  size?: number;
  className?: string;
};

export function BrandLogo({ domain, name, size = 32, className }: BrandLogoProps) {
  const [failedDomain, setFailedDomain] = useState<string | null>(null);

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
      className={cn(
        "bg-background ring-border shrink-0 rounded-lg object-contain p-1 ring-1",
        className,
      )}
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
        "bg-muted text-muted-foreground ring-border flex shrink-0 items-center justify-center rounded-lg font-semibold ring-1 select-none",
        className,
      )}
      style={{ width: size, height: size, fontSize: Math.max(10, Math.round(size * 0.38)) }}
    >
      {getNameInitials(name, "AI", { maxLength: 2 })}
    </span>
  );
}
