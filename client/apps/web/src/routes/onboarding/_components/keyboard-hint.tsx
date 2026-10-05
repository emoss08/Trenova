import type { ReactNode } from "react";

export function Kbd({ children }: { children: ReactNode }) {
  return <span className="nv-kbd">{children}</span>;
}
