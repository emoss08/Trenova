import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { PlusIcon } from "lucide-react";

/**
 * The shape of the page once policies exist: a row of cards, each with the
 * parts a real one has. The document mark, the title and its summary, the
 * version and audience pills, and the signed-by line along the bottom.
 */
const GHOST_CARDS: readonly { title: string; summary: string }[] = [
  { title: "w-3/4", summary: "w-full" },
  { title: "w-1/2", summary: "w-4/5" },
  { title: "w-2/3", summary: "w-3/5" },
];

type PoliciesEmptyProps = {
  title: string;
  description: string;
  onPublish?: () => void;
  className?: string;
};

export function PoliciesEmpty({ title, description, onPublish, className }: PoliciesEmptyProps) {
  const t = useT();

  return (
    <EmptySheet
      className={className}
      sketchClassName="max-w-2xl"
      title={title}
      description={description}
      action={
        onPublish ? (
          <Button variant="outline" size="sm" onClick={onPublish}>
            <PlusIcon className="size-3.5" />
            {t("Publish a policy")}
          </Button>
        ) : null
      }
      sketch={
        <div className="grid gap-3 sm:grid-cols-3">
          {GHOST_CARDS.map((card, index) => (
            <div
              key={index}
              className="border-border/70 bg-card flex flex-col gap-3 rounded-lg border p-3"
            >
              <div className="flex items-start gap-3">
                <span className="bg-muted size-7 shrink-0 rounded-md" />
                <div className="flex min-w-0 flex-1 flex-col gap-2 pt-1">
                  <GhostLine className={`h-2 ${card.title}`} />
                  <GhostLine className={card.summary} />
                </div>
              </div>
              <div className="flex items-center gap-1.5">
                <GhostPill className="w-8" />
                <GhostPill className="w-14" />
                <GhostPill className="w-12" />
              </div>
              <div className="border-border/60 mt-auto flex items-center border-t border-dashed pt-3">
                <GhostPill className="h-5 w-20 rounded-md" />
              </div>
            </div>
          ))}
        </div>
      }
    />
  );
}

function GhostPill({ className }: { className?: string }) {
  return <span className={`border-border/70 block h-4 rounded-md border ${className ?? ""}`} />;
}
