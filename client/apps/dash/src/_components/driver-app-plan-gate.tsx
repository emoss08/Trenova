import { Lock01Icon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { useDriverAppOffPlan } from "@/lib/plan-handler";
import { useId, type ReactNode } from "react";

/** The whole screen a driver sees when the company's plan does not include the driver app. */
export function DriverAppOffPlan() {
  const t = useT();
  const titleId = useId();

  return (
    <main
      aria-labelledby={titleId}
      data-slot="driver-app-off-plan"
      className="bg-canvas text-foreground flex min-h-dvh flex-col items-center justify-center px-6 text-center"
    >
      <span
        aria-hidden
        className="border-neutral-border bg-neutral-subtle text-neutral-subtle-foreground flex size-12 items-center justify-center rounded-lg border"
      >
        <Lock01Icon className="size-5" />
      </span>
      <h1 id={titleId} className="mt-4 max-w-sm text-xl font-semibold text-balance">
        {t("The driver app is not part of your company's plan.")}
      </h1>
      <p className="text-foreground-muted mt-2 text-base">{t("Ask your dispatcher.")}</p>
    </main>
  );
}

/** Renders the app until the server says the driver app is not on the company's plan. */
export function DriverAppPlanGate({ children }: { children: ReactNode }) {
  const offPlan = useDriverAppOffPlan();
  return offPlan ? <DriverAppOffPlan /> : <>{children}</>;
}
