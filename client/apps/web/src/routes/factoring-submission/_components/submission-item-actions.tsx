import { factoringItemActions } from "@/lib/factoring";
import type { FactoringReserveAction, FactoringSubmissionItem } from "@/lib/graphql/factoring";
import {
  CoinsHandIcon,
  DotsHorizontalIcon,
  FlipBackwardIcon,
  ReverseLeftIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useT } from "@trenova/shared/i18n/use-t";

/**
 * The movements one invoice on a schedule can take next: a funded invoice's
 * reserve is released or (under recourse) charged back, and a charged-back
 * invoice's chargeback can be reversed. Renders nothing when none applies.
 */
export function SubmissionItemActions({
  item,
  recourse,
  onSelect,
}: {
  item: FactoringSubmissionItem;
  recourse: boolean;
  onSelect: (action: FactoringReserveAction) => void;
}) {
  const t = useT();
  const actions = factoringItemActions(item.status, recourse);

  if (actions.length === 0) {
    return null;
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            type="button"
            size="icon-sm"
            variant="ghost"
            aria-label={t("Actions for invoice {0}", item.invoiceNumber)}
          >
            <DotsHorizontalIcon className="size-4" />
          </Button>
        }
      />
      <DropdownMenuContent align="end" className="w-56">
        {actions.map((action) => {
          switch (action) {
            case "release":
              return (
                <DropdownMenuItem
                  key={action}
                  title={t("Release reserve")}
                  startContent={<CoinsHandIcon className="size-3.5" />}
                  onClick={() => onSelect(action)}
                />
              );
            case "chargeback":
              return (
                <DropdownMenuItem
                  key={action}
                  title={t("Charge back")}
                  color="danger"
                  startContent={<ReverseLeftIcon className="size-3.5" />}
                  onClick={() => onSelect(action)}
                />
              );
            case "reverse":
              return (
                <DropdownMenuItem
                  key={action}
                  title={t("Reverse chargeback")}
                  startContent={<FlipBackwardIcon className="size-3.5" />}
                  onClick={() => onSelect(action)}
                />
              );
          }
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
