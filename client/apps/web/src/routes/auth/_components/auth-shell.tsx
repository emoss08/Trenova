import logo from "@/assets/logo.webp";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";
import { LegalAgreementNote } from "./legal-agreement-note";
import { AuthStage, AuthStageCanvas } from "./stage/auth-stage";

// The step the sign-in flow is on.
export type AuthStep = "login" | "forgot" | "org" | "role" | "done";

/**
 * The sign-in frame: the form column on the left — brand, the screen, the legal line —
 * and the shader stage on the right. The screen is centred by auto margins rather than
 * spaced between its neighbours, so it stays put when the install publishes no legal
 * line. At 860px and below it is one column with the stage as a 200px band on top, and
 * the whole frame scrolls instead of the column.
 *
 * The screen is keyed by `screenKey`, so moving between steps replays its entrance.
 *
 * The `minmax(0,…)` tracks and `scrollbar-gutter:stable` are load-bearing together —
 * an auto-sized track grows a horizontal scrollbar the moment the tallest screen makes
 * the column scroll vertically.
 */
export function AuthShell({ screenKey, children }: { screenKey: string; children: ReactNode }) {
  const t = useT();

  return (
    <AuthStage>
      <div
        data-auth-shell=""
        className="bg-auth-canvas font-geist text-foreground fixed inset-0 grid w-full grid-cols-[minmax(0,1fr)] grid-rows-[200px_minmax(0,auto)] overflow-auto tracking-[-0.005em] antialiased min-[861px]:h-svh min-[861px]:grid-cols-[minmax(380px,520px)_minmax(0,1fr)] min-[861px]:grid-rows-[minmax(0,1fr)] min-[861px]:overflow-hidden"
      >
        <main className="relative z-1 row-start-2 flex min-w-0 flex-col px-[clamp(28px,5vw,64px)] py-8 min-[861px]:row-start-1 min-[861px]:[scrollbar-gutter:stable] min-[861px]:overflow-auto">
          <div className="flex items-center gap-2.5 self-start">
            <img src={logo} alt="" className="size-6 object-contain" />
            <span className="text-lg font-semibold tracking-[-0.02em]">{t("Trenova")}</span>
          </div>
          <div key={screenKey} className="auth-enter my-auto flex w-full max-w-[360px] flex-col py-12">
            {children}
          </div>
          <LegalAgreementNote />
        </main>
        <div className="border-border relative row-start-1 min-w-0 overflow-hidden border-b min-[861px]:col-start-2 min-[861px]:border-b-0 min-[861px]:border-l">
          <AuthStageCanvas />
        </div>
      </div>
    </AuthStage>
  );
}
