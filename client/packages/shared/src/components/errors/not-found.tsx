import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowLeftIcon, HouseIcon } from "lucide-react";
import { useId } from "react";
import { StatusScreen } from "./status-screen";

// RouteOffMap is a lane that leaves its origin and ends at a stop that is not there. It is
// decoration only; the heading says the same thing in words.
function RouteOffMap({ className }: { className?: string }) {
  return (
    <svg
      aria-hidden
      viewBox="0 0 240 72"
      fill="none"
      className={cn("h-18 w-60", className)}
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path
        d="M20 52 C 56 52, 60 20, 100 20 S 150 50, 184 38"
        className="text-border-strong"
        stroke="currentColor"
        strokeWidth="2"
        strokeDasharray="1 7"
      />
      <circle cx="20" cy="52" r="6" className="text-foreground" fill="currentColor" />
      <circle
        cx="20"
        cy="52"
        r="11"
        className="text-border"
        stroke="currentColor"
        strokeWidth="1.5"
      />
      <circle
        cx="208"
        cy="30"
        r="14"
        className="text-foreground-subtle"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeDasharray="3 4"
      />
      <path
        d="M203.5 25.5 C 203.5 22.5, 206 21, 208 21 C 210.5 21, 212.5 22.8, 212.5 25.2 C 212.5 28.4, 208 28.6, 208 32"
        className="text-foreground-subtle"
        stroke="currentColor"
        strokeWidth="1.75"
      />
      <circle cx="208" cy="36.5" r="1.1" className="text-foreground-subtle" fill="currentColor" />
    </svg>
  );
}

export type NotFoundStateProps = {
  path?: string;
  onGoHome: () => void;
  homeLabel?: string;
  // onGoBack is offered only when there is somewhere to go back to inside the app.
  onGoBack?: () => void;
  headingLevel?: "h1" | "h2";
  className?: string;
};

export function NotFoundState({
  path,
  onGoHome,
  homeLabel,
  onGoBack,
  headingLevel = "h2",
  className,
}: NotFoundStateProps) {
  const t = useT();
  const titleId = useId();
  const Heading = headingLevel;

  return (
    <section
      aria-labelledby={titleId}
      data-slot="not-found"
      className={cn(
        "animate-rise flex w-full flex-1 flex-col items-center justify-center px-4 py-16 text-center",
        className,
      )}
    >
      <RouteOffMap />
      <p className="text-foreground-subtle mt-6 font-mono text-xs tabular-nums">{t("Error 404")}</p>
      <Heading id={titleId} className="mt-1.5 text-2xl font-semibold text-balance">
        {t("We can't find that page")}
      </Heading>
      <p className="text-foreground-muted mt-2 max-w-[48ch] text-base text-pretty">
        {t(
          "The address may be mistyped, or the page may have moved or been removed. Check the link, or head back to somewhere you know.",
        )}
      </p>
      {path ? (
        <code
          className="bg-sunken border-border-subtle text-foreground-muted mt-4 max-w-full truncate rounded-md border px-2 py-1 font-mono text-xs"
          title={path}
        >
          {path}
        </code>
      ) : null}
      <div className="mt-6 flex flex-wrap items-center justify-center gap-2">
        <Button onClick={onGoHome}>
          <HouseIcon />
          {homeLabel ?? t("Go to dashboard")}
        </Button>
        {onGoBack ? (
          <Button variant="outline" onClick={onGoBack}>
            <ArrowLeftIcon />
            {t("Go back")}
          </Button>
        ) : null}
      </div>
    </section>
  );
}

// NotFoundPage takes the whole window, for an address that fell outside every layout the
// app has. Inside the app shell, NotFoundState renders in the page area instead so the
// navigation stays where it was.
export function NotFoundPage(props: Omit<NotFoundStateProps, "headingLevel">) {
  const t = useT();

  return (
    <StatusScreen meta={t("404 · Not found")}>
      <NotFoundState {...props} headingLevel="h1" />
    </StatusScreen>
  );
}
