import { useT } from "@trenova/shared/i18n/use-t";
import { SUPPORT_EMAIL } from "@trenova/shared/lib/constants";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

type StatusScreenProps = {
  children: ReactNode;
  // meta sits at the foot opposite the support line: the status code, a reference.
  meta?: ReactNode;
  className?: string;
  // supportEmail is where the foot sends people for help; the build's VITE_SUPPORT_EMAIL
  // by default. Empty leaves the line out: an install that names no address has none.
  supportEmail?: string;
};

// StatusScreen is the frame for a state that has taken the whole window — a 404 outside the
// app, a crash above the router. It keeps the product's name at the top and, when the
// install names one, a way to reach a person at the foot, and puts nothing else around the
// state itself.
export function StatusScreen({
  children,
  meta,
  className,
  supportEmail = SUPPORT_EMAIL,
}: StatusScreenProps) {
  const t = useT();
  const hasFooter = supportEmail !== "" || Boolean(meta);

  return (
    <div
      data-slot="status-screen"
      className={cn("bg-canvas text-foreground flex min-h-dvh flex-col", className)}
    >
      <header className="border-border-subtle flex h-12 shrink-0 items-center border-b px-6">
        <a
          href="/"
          className="ui-focus-ring rounded-sm text-sm font-semibold"
          aria-label={t("Trenova home")}
        >
          {t("Trenova")}
        </a>
      </header>
      <main className="flex flex-1 flex-col items-center justify-center">{children}</main>
      {hasFooter ? (
        <footer className="border-border-subtle text-foreground-subtle flex shrink-0 flex-wrap items-center justify-between gap-2 border-t px-6 py-3 text-xs">
          {supportEmail ? (
            <span>
              {t("Need help?")}{" "}
              <a
                href={`mailto:${supportEmail}`}
                className="ui-focus-ring text-brand rounded-sm underline-offset-4 hover:underline"
              >
                {supportEmail}
              </a>
            </span>
          ) : (
            <span />
          )}
          {meta ? <span className="font-mono tabular-nums">{meta}</span> : null}
        </footer>
      ) : null}
    </div>
  );
}
