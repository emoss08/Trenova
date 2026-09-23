import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { WorkingDot } from "./voice/working-dot";

/**
 * Said in a conversation's row while its reply is still being written, in
 * place of when it was last touched: the reply is the newer fact, and opening
 * the conversation picks it up where it has got to.
 *
 * The words are the label, so a screen reader hears the row as "…, writing a
 * reply" with nothing extra to announce; the dot is the same still mark the
 * rest of the product uses for work under way.
 */
export function LiveReplyLabel({ className }: { className?: string }) {
  const t = useT();

  return (
    <span className={cn("text-foreground inline-flex items-center gap-1.5", className)}>
      <WorkingDot working still />
      {t("Writing a reply")}
    </span>
  );
}
