import { Button } from "./ui/button";

type NotFoundPageProps = {
  onGoHome: () => void;
  isDev: boolean;
  errorName?: string;
  errorMessage?: string;
  path?: string;
};

export function NotFoundPage({
  onGoHome,
  isDev,
  errorName,
  errorMessage,
  path,
}: NotFoundPageProps) {
  return (
    <div className="relative min-h-screen overflow-hidden bg-background text-foreground">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_top,hsl(var(--foreground)/0.12),transparent_52%)] dark:bg-[radial-gradient(ellipse_at_top,hsl(var(--foreground)/0.08),transparent_52%)]" />
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(to_bottom,hsl(var(--foreground)/0.03)_1px,transparent_1px)] bg-size-[100%_8px] opacity-30 dark:opacity-20" />
      <div className="relative mx-auto flex min-h-screen w-full max-w-6xl flex-col px-6 pt-7 pb-8 sm:px-10 sm:pt-8">
        <main className="relative z-10 flex flex-1 flex-col items-center justify-center pt-12 pb-28 text-center sm:pt-16 sm:pb-36">
          <p className="text-[7rem] leading-none font-semibold tracking-[-0.04em] sm:text-[9rem] md:text-[11rem]">
            404
          </p>
          <h1 className="mt-5 text-2xl font-semibold sm:text-3xl">
            It seems you got a little bit lost
          </h1>
          <p className="mx-auto mt-4 max-w-2xl text-sm text-muted-foreground sm:text-base">
            The destination you requested does not exist, may have moved, or the URL might be
            incorrect.
          </p>

          <div className="mt-9 flex w-full max-w-xl flex-wrap items-center justify-center gap-3">
            <Button onClick={onGoHome} variant="link" className="min-w-36">
              Go back to homepage
            </Button>
          </div>
        </main>

        <div className="pointer-events-none absolute bottom-110 left-1/2 h-px w-[88%] -translate-x-1/2 bg-linear-to-r from-transparent via-foreground/35 to-transparent blur-[1px]" />

        <footer className="relative z-10 pt-5">
          {isDev && errorName && errorMessage && (
            <div className="mb-5 rounded-xl border border-border/70 bg-background/60 p-4 text-left">
              <p className="text-xs font-semibold tracking-wide text-muted-foreground uppercase">
                Development Details
              </p>
              <p className="mt-2 font-mono text-xs text-destructive">
                {errorName}: {errorMessage}
              </p>
              {path && <p className="mt-2 text-xs text-muted-foreground">Path: {path}</p>}
            </div>
          )}

          <div className="grid gap-5 text-left text-sm text-muted-foreground sm:grid-cols-3">
            <div>
              <p className="text-[11px] tracking-wide uppercase">Need Help?</p>
              <p className="mt-2 text-base font-medium text-foreground">support@trenova.com</p>
            </div>
            <div>
              <p className="text-[11px] tracking-wide uppercase">Quick Links</p>
              <p className="mt-2">Dashboard</p>
              <p>Shipments</p>
            </div>
            <div className="sm:text-right">
              <p className="text-[11px] tracking-wide uppercase">Navigation</p>
              <p className="mt-2">Go back to top</p>
            </div>
          </div>
        </footer>
      </div>
    </div>
  );
}
