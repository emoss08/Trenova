import type { ReactNode } from "react";
import { useMorphHeight } from "./auth-primitives";

/**
 * The single card every step lives in. Its height is animated to whatever the active
 * step measures, and while a step is pending a conic-gradient beam sweeps its border —
 * the only progress affordance on a screen that is otherwise waiting on the network.
 * The beam keys off the pending button inside the card (see .auth-card in app.css).
 */
export function AuthCard({ stepKey, children }: { stepKey: string; children: ReactNode }) {
  const { outerRef, innerRef, animated } = useMorphHeight();

  return (
    <div className="auth-card bg-card border-border relative rounded-[14px] border shadow-[var(--auth-card-shadow)]">
      <span className="auth-beam" aria-hidden="true" />
      <div className="overflow-hidden rounded-[14px]">
        <div ref={outerRef} className={animated ? "auth-card-height" : undefined}>
          <div ref={innerRef}>
            <div key={stepKey} className="auth-step-enter">
              {children}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export function AuthCardBody({ children }: { children: ReactNode }) {
  return <div className="p-[22px]">{children}</div>;
}
