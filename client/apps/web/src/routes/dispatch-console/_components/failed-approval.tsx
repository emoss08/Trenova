import { recordPath } from "@/config/record-links";
import type { DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import { AlertTriangleIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
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
        <Button
          variant="link"
          size="xxs"
          className="px-0"
          nativeButton={false}
          render={<Link to={recordPath("shipment", move.shipmentId)} />}
        >
          {t("Open shipment")}
        </Button>
      </AlertDescription>
    </Alert>
  );
}
