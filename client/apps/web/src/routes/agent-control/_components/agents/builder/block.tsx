import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { useController, useFormContext, type FieldPath } from "react-hook-form";
import type { AgentFormValues } from "../agent-form-schema";

export type BlockId = "who" | "instr" | "trig" | "tools" | "limits" | "team" | "record";

type BlkProps = {
  id: BlockId;
  title: string;
  note?: string;
  /** Glows briefly after Nova filled it. */
  fresh?: boolean;
  actions?: ReactNode;
  children: ReactNode;
};

/** One titled part of the builder's canvas. */
export function Blk({ id, title, note, fresh = false, actions, children }: BlkProps) {
  return (
    <section id={`ab-${id}`} className={cn("blk", fresh && "fresh")} aria-labelledby={`ab-${id}-h`}>
      <header className="blk-h">
        <div>
          <h2 id={`ab-${id}-h`}>{title}</h2>
          {note && <p>{note}</p>}
        </div>
        {actions}
      </header>
      {children}
    </section>
  );
}

/** One of the draft's values and a way to change it, marking the draft changed. */
export function useDraftField<K extends FieldPath<AgentFormValues>>(name: K) {
  const { control } = useFormContext<AgentFormValues>();
  const { field } = useController({ control, name });
  return [field.value, field.onChange] as const;
}
