import { CheckIcon } from "@trenova/shared/components/icons";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import type { QueueLeave } from "./use-queue-cursor";

/** The card the queue's current item is read in, and how it leaves. */
export function QueueCard({
  leaving,
  className,
  children,
}: {
  leaving: QueueLeave | null;
  className?: string;
  children: ReactNode;
}) {
  return (
    <div className={cn("dk-dc2-card", leaving && `dk-out-${leaving}`, className)}>{children}</div>
  );
}

/** What the middle shows once nothing is left to work. */
export function QueueClear({ title, body }: { title: string; body: string }) {
  return (
    <div className="dk-dc2-clear">
      <span className="dk-dc2-cic">
        <CheckIcon className="size-5.5" strokeWidth={2.2} />
      </span>
      <b>{title}</b>
      <span>{body}</span>
    </div>
  );
}

/** A quiet line in the middle while the queue is first read. */
export function QueueReading({ label }: { label: string }) {
  return (
    <div className="dk-dc2-clear" aria-busy="true">
      <span className="dk-dc2-qe">{label}</span>
    </div>
  );
}
