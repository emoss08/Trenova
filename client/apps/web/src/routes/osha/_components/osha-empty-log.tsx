import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostBox, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { cn } from "@trenova/shared/lib/utils";
import { Link } from "react-router";
import { FormMark } from "./osha-form-marks";

const GHOST_ROWS = 3;
const GRID =
  "grid grid-cols-[2.25rem_minmax(0,1fr)_3rem_minmax(0,1.2fr)_5.5rem_3.5rem] items-center gap-3";

type OshaEmptyLogProps = {
  title: string;
  description: string;
  action?: { label: string; to: string } | { label: string; onClick: () => void };
  className?: string;
};

/**
 * An empty log drawn as what it is: a ruled page of the 300 with nothing
 * written on it. The column marks are the form's own, the lines fade out
 * where the entries would go, and the words underneath say what to do next.
 */
export function OshaEmptyLog({ title, description, action, className }: OshaEmptyLogProps) {
  return (
    <EmptySheet
      className={className}
      title={title}
      description={description}
      action={
        action ? (
          "to" in action ? (
            <Button variant="outline" size="sm" render={<Link to={action.to} />}>
              {action.label}
            </Button>
          ) : (
            <Button variant="outline" size="sm" onClick={action.onClick}>
              {action.label}
            </Button>
          )
        ) : null
      }
      sketch={
        <div className="border-border/70 bg-card rounded-md border">
          <div
            className={cn(
              GRID,
              "text-muted-foreground border-b px-3 py-1.5 text-left text-2xs leading-none",
            )}
          >
            <span>Case</span>
            <span>Employee</span>
            <span>Date</span>
            <span>What happened</span>
            <span className="flex gap-1">
              <FormMark>G</FormMark>
              <FormMark>H</FormMark>
              <FormMark>I</FormMark>
              <FormMark>J</FormMark>
            </span>
            <span className="flex gap-1">
              <FormMark>K</FormMark>
              <FormMark>L</FormMark>
            </span>
          </div>
          {Array.from({ length: GHOST_ROWS }, (_, index) => (
            <div
              key={index}
              className={cn(
                GRID,
                "border-border/60 border-b border-dashed px-3 py-2.5 last:border-0",
              )}
            >
              <GhostLine className="w-5" />
              <GhostLine className="w-3/5" />
              <GhostLine className="w-8" />
              <GhostLine className="w-4/5" />
              <span className="flex gap-1">
                {Array.from({ length: 4 }, (_, box) => (
                  <GhostBox key={box} />
                ))}
              </span>
              <span className="flex gap-1">
                <GhostBox />
                <GhostBox />
              </span>
            </div>
          ))}
        </div>
      }
    />
  );
}
