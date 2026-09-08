import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { cn } from "@trenova/shared/lib/utils";
import { XIcon } from "lucide-react";

/**
 * A billing workspace list as it will look with records in it: a card per
 * record with the number and the amount on the first line, the customer
 * under it, and the status pill and the age along the bottom, the way every
 * billing record card reads.
 */
const GHOST_CARDS: readonly { number: string; customer: string; amount: string; age: string }[] = [
  { number: "w-24", customer: "w-3/5", amount: "w-12", age: "w-10" },
  { number: "w-20", customer: "w-2/5", amount: "w-14", age: "w-8" },
  { number: "w-28", customer: "w-1/2", amount: "w-10", age: "w-12" },
];

type BillingListEmptyProps = {
  title: string;
  description: string;
  /** Offered when a search or filter is what emptied the list. */
  onClearFilters?: () => void;
  className?: string;
};

export function BillingListEmpty({
  title,
  description,
  onClearFilters,
  className,
}: BillingListEmptyProps) {
  return (
    <EmptySheet
      className={cn("py-6", className)}
      sketchClassName="max-w-xs"
      title={title}
      description={description}
      action={
        onClearFilters ? (
          <Button variant="outline" size="sm" onClick={onClearFilters}>
            <XIcon className="size-3.5" />
            Clear filters
          </Button>
        ) : null
      }
      sketch={
        <div className="flex flex-col gap-1.5 text-left">
          {GHOST_CARDS.map((card, index) => (
            <div
              key={index}
              className="border-border/60 bg-card flex flex-col gap-2 rounded-lg border p-3"
            >
              <div className="flex items-center justify-between gap-2">
                <GhostLine className={`h-2 ${card.number}`} />
                <GhostLine className={`h-2 ${card.amount}`} />
              </div>
              <GhostLine className={card.customer} />
              <div className="flex items-center justify-between gap-2 pt-0.5">
                <GhostPill className="w-14" />
                <GhostLine className={card.age} />
              </div>
            </div>
          ))}
        </div>
      }
    />
  );
}

const GHOST_ROWS: readonly { name: string; amount: string }[] = [
  { name: "w-32", amount: "w-14" },
  { name: "w-24", amount: "w-10" },
  { name: "w-28", amount: "w-12" },
];

const GHOST_FACTS = 3;

type BillingDetailUnselectedProps = {
  title: string;
  description: string;
  /**
   * The shape of the pane it stands in for: a tabbed record with charge lines
   * under the tab strip, or a review laid out as cards of facts and figures.
   */
  layout: "tabs" | "cards";
  className?: string;
};

/**
 * A billing detail pane as it will look with a record open: the number and
 * its status, the amount beside the customer, a row of facts, then either
 * the actions, tabs and first lines of a record, or the cards of a review.
 */
export function BillingDetailUnselected({
  title,
  description,
  layout,
  className,
}: BillingDetailUnselectedProps) {
  return (
    <EmptySheet
      className={cn("h-full justify-center", className)}
      title={title}
      description={description}
      sketch={
        <div className="border-border/70 bg-card flex flex-col rounded-md border text-left">
          <div className="flex flex-col gap-3 border-b px-4 py-3">
            <div className="flex items-center gap-2">
              <GhostLine className="h-2.5 w-24" />
              <GhostPill className="w-16" />
            </div>
            <div className="flex items-baseline gap-3">
              <GhostLine className="h-3.5 w-20" />
              <GhostLine className="w-28" />
            </div>
            <div className="grid grid-cols-3 gap-x-6 gap-y-2">
              {Array.from({ length: GHOST_FACTS }, (_, index) => (
                <div key={index} className="flex flex-col gap-1.5">
                  <GhostLine className="w-10" />
                  <GhostLine className="h-2 w-16" />
                </div>
              ))}
            </div>
          </div>
          {layout === "tabs" ? <GhostTabbedRecord /> : <GhostReviewCards />}
        </div>
      }
    />
  );
}

function GhostTabbedRecord() {
  return (
    <>
      <div className="flex items-center gap-2 border-b px-4 py-2">
        <GhostPill className="h-5 w-16 rounded-md" />
        <GhostPill className="h-5 w-14 rounded-md" />
        <GhostPill className="h-5 w-12 rounded-md" />
      </div>
      <div className="flex items-center gap-4 border-b px-4 py-2">
        <GhostLine className="bg-foreground/40 h-2 w-12" />
        <GhostLine className="w-14" />
        <GhostLine className="w-14" />
        <GhostLine className="w-12" />
      </div>
      <div className="divide-border/60 flex flex-col divide-y divide-dashed px-4">
        {GHOST_ROWS.map((row, index) => (
          <div key={index} className="flex items-center justify-between gap-3 py-2.5">
            <GhostLine className={`h-2 ${row.name}`} />
            <GhostLine className={`h-2 ${row.amount}`} />
          </div>
        ))}
      </div>
    </>
  );
}

function GhostReviewCards() {
  return (
    <div className="grid grid-cols-2 gap-3 p-3">
      <div className="border-border/60 flex flex-col gap-2.5 rounded-md border p-3">
        <GhostLine className="w-16" />
        {GHOST_ROWS.map((row, index) => (
          <div key={index} className="flex items-center justify-between gap-3">
            <GhostLine className={row.name} />
            <GhostLine className={`h-2 ${row.amount}`} />
          </div>
        ))}
      </div>
      <div className="border-border/60 flex flex-col gap-2.5 rounded-md border p-3">
        <GhostLine className="w-14" />
        <div className="grid grid-cols-2 gap-x-4 gap-y-2">
          {Array.from({ length: 4 }, (_, index) => (
            <div key={index} className="flex flex-col gap-1.5">
              <GhostLine className="w-10" />
              <GhostLine className="h-2 w-14" />
            </div>
          ))}
        </div>
        <div className="mt-auto flex gap-1.5 pt-1">
          <GhostPill className="h-5 w-14 rounded-md" />
          <GhostPill className="h-5 w-12 rounded-md" />
        </div>
      </div>
    </div>
  );
}

function GhostPill({ className }: { className?: string }) {
  return <span className={cn("border-border/70 block h-4 rounded-full border", className)} />;
}
