import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

const SUPPORT_EMAIL = "support@trenova.com";

type StatusScreenProps = {
  children: ReactNode;
  // meta sits at the foot opposite the support line: the status code, a reference.
  meta?: ReactNode;
  className?: string;
};

// StatusScreen is the frame for a state that has taken the whole window — a 404 outside the
// app, a crash above the router. It keeps the product's name at the top and a way to reach
// a person at the foot, and puts nothing else around the state itself.
export function StatusScreen({ children, meta, className }: StatusScreenProps) {
  const t = useT();

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
      <footer className="border-border-subtle text-foreground-subtle flex shrink-0 flex-wrap items-center justify-between gap-2 border-t px-6 py-3 text-xs">
        <span>
          {t("Need help?")}{" "}
          <a
            href={`mailto:${SUPPORT_EMAIL}`}
            className="ui-focus-ring text-brand rounded-sm underline-offset-4 hover:underline"
          >
            {SUPPORT_EMAIL}
          </a>
        </span>
        {meta ? <span className="font-mono tabular-nums">{meta}</span> : null}
      </footer>
    </div>
  );
}
