import { createContext, use, type ReactNode } from "react";

const WorkerStripContext = createContext<ReactNode>(null);

/**
 * Hands a tab the section strip the shell built for it, for a tab that draws content above its
 * strip (Employment's lifecycle route). The shell owns the strip and its state lines; the tab only
 * chooses where it goes, by placing `WorkerSectionStrip`. Tabs the shell does not list as placing
 * their own strip get it drawn above them and never read this.
 */
export function WorkerStripProvider({
  strip,
  children,
}: {
  strip: ReactNode;
  children: ReactNode;
}) {
  return <WorkerStripContext value={strip}>{children}</WorkerStripContext>;
}

/** Where the tab wants its section strip; nothing when the tab has a single view. */
export function WorkerSectionStrip() {
  return use(WorkerStripContext);
}
