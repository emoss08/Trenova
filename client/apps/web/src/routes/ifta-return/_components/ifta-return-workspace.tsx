import { useT } from "@trenova/shared/i18n/use-t";
import type { IftaPeriodKey } from "@/lib/ifta-return";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { TriangleAlertIcon } from "lucide-react";
import { IftaReturnEmpty } from "./ifta-return-empty";
import { IftaReturnSkeleton } from "./ifta-return-skeleton";
import { useGenerateIftaReturn } from "./mutations";
import { useIftaReturnPermissions } from "./permissions";
import { QuarterPicker } from "./quarter-picker";
import { iftaPeriodQuery, iftaReturnForPeriodQuery } from "./queries";
import { ReturnActions } from "./return-actions";
import { ReturnDiagnostics } from "./return-diagnostics";
import { ReturnLinesTable } from "./return-lines-table";
import { ReturnSummaryStrip } from "./return-summary-strip";

type IftaReturnWorkspaceProps = {
  period: IftaPeriodKey;
  onPeriodChange: (period: IftaPeriodKey) => void;
};

export function IftaReturnWorkspace({ period, onPeriodChange }: IftaReturnWorkspaceProps) {
  const t = useT();

  const perms = useIftaReturnPermissions();
  const returnQuery = useQuery(iftaReturnForPeriodQuery(period));
  const periodQuery = useQuery(iftaPeriodQuery(period));
  const generate = useGenerateIftaReturn(period);

  const ret = returnQuery.data ?? null;

  return (
    <div className="flex flex-col gap-4">
      <QuarterPicker
        period={period}
        onPeriodChange={onPeriodChange}
        status={ret?.status ?? null}
        amendmentNumber={ret?.amendmentNumber ?? 0}
        detail={ret?.period ?? periodQuery.data ?? null}
      />

      {returnQuery.isPending ? (
        <IftaReturnSkeleton />
      ) : returnQuery.isError ? (
        <Alert variant="destructive">
          <TriangleAlertIcon className="size-4" />
          <AlertTitle>{t("The return could not be read")}</AlertTitle>
          <AlertDescription className="flex flex-col items-start gap-2">
            <span>
              {t("{0} Nothing was changed.", returnQuery.error instanceof Error
                ? returnQuery.error.message
                : "Something went wrong reading the quarter.")}
            </span>
            <Button
              variant="outline"
              size="sm"
              onClick={() => void returnQuery.refetch()}
              disabled={returnQuery.isFetching}
            >
              {returnQuery.isFetching ? "Retrying..." : "Retry"}
            </Button>
          </AlertDescription>
        </Alert>
      ) : ret ? (
        <>
          <div className="flex justify-end">
            <ReturnActions ret={ret} period={period} perms={perms} />
          </div>
          <ReturnSummaryStrip ret={ret} />
          <ReturnLinesTable ret={ret} />
          <ReturnDiagnostics ret={ret} canBackfill={perms.manage} />
        </>
      ) : (
        <IftaReturnEmpty
          period={period}
          canCreate={perms.create}
          onGenerate={() => generate.mutate()}
          isGenerating={generate.isPending}
        />
      )}
    </div>
  );
}

export default IftaReturnWorkspace;
