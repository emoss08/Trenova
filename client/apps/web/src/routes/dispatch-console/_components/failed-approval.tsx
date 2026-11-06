import { recordPath } from "@/config/record-links";
import type { DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import { AlertTriangleIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { buttonVariants } from "@trenova/shared/lib/variants/button";
import { Link } from "react-router";

type FailedProposal = NonNullable<DispatchBoardMove["failedProposal"]>;

/**
 * An approval of Nova's driver that could not be carried out: who was approved, when, and
 * why the assignment was refused, with the shipment one click away to fix what refused it.
 * It stays until the move changes or a newer proposal replaces it, so the move never looks
 * as though nobody had decided on it.
 */
export function FailedApproval({
  move,
  failed,
}: {
  move: Pick<DispatchBoardMove, "shipmentId">;
  failed: FailedProposal;
}) {
  const t = useT();

  return (
    <Alert size="sm" variant="warning">
      <AlertTriangleIcon aria-hidden />
      <AlertTitle>
        {failed.workerName
          ? t(
              "Approved {0} at {1}, but it wasn't assigned",
              failed.workerName,
              formatUnixTime(failed.failedAt),
            )
          : t("Approved at {0}, but it wasn't assigned", formatUnixTime(failed.failedAt))}
      </AlertTitle>
      <AlertDescription className="flex flex-col items-start gap-1.5">
        <span>{failed.reason}</span>
        <span className="text-muted-foreground">
          {t("Fix it on the shipment, then assign a driver below or propose coverage again.")}
        </span>
        <Link
          to={recordPath("shipment", move.shipmentId)}
          className={cn(buttonVariants({ variant: "link", size: "xxs" }), "px-0")}
        >
          {t("Open shipment")}
        </Link>
      </AlertDescription>
    </Alert>
  );
}
