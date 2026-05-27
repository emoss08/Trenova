import type { ReactNode } from "react";

export function OuterSection({ children }: { children: ReactNode }) {
  return <section className="space-y-4 pb-10">{children}</section>;
}

export function OuterContent({ children }: { children: ReactNode }) {
  return <div className="space-y-3">{children}</div>;
}
