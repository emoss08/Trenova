import { useT } from "@trenova/shared/i18n/use-t";
import { buildCsv, downloadCsv } from "@/lib/data-table-export";
import {
  canFinalize,
  IFTA_RETURN_CSV_COLUMNS,
  iftaReturnCsvFilename,
  iftaReturnCsvRows,
  linesMissingRates,
  returnTransitions,
  type IftaPeriodKey,
  type IftaReturnTransition,
  type IftaReturnView,
} from "@/lib/ifta-return";
import { Button } from "@trenova/shared/components/ui/button";
import { pluralize } from "@trenova/shared/lib/utils";
import {
  DownloadIcon,
  FilePlus2Icon,
  LockIcon,
  LockOpenIcon,
  PlayIcon,
  RefreshCwIcon,
  SendIcon,
  Trash2Icon,
} from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { AmendReturnDialog } from "./amend-return-dialog";
import { DeleteReturnDialog } from "./delete-return-dialog";
import { FinalizeReturnDialog } from "./finalize-return-dialog";
import { MarkFiledDialog } from "./mark-filed-dialog";
import { useGenerateIftaReturn, useRecomputeIftaReturn } from "./mutations";
import type { IftaReturnWorkspacePermissions } from "./permissions";
import { ReopenReturnDialog } from "./reopen-return-dialog";

type OpenDialog = "finalize" | "reopen" | "markFiled" | "amend" | "delete" | null;

type ReturnActionsProps = {
  ret: IftaReturnView | null;
  period: IftaPeriodKey;
  perms: IftaReturnWorkspacePermissions;
};

export function ReturnActions({ ret, period, perms }: ReturnActionsProps) {
  const t = useT();

  const [dialog, setDialog] = useState<OpenDialog>(null);
  const generate = useGenerateIftaReturn(period);
  const recompute = useRecomputeIftaReturn(period);

  const transitions = new Set<IftaReturnTransition>(returnTransitions(ret?.status ?? null, perms));
  const missingRates = ret ? linesMissingRates(ret.lines) : [];
  const finalizeBlocked = ret === null || !canFinalize(ret);

  const onExport = () => {
    if (!ret) return;
    downloadCsv(
      buildCsv(iftaReturnCsvRows(ret), IFTA_RETURN_CSV_COLUMNS),
      iftaReturnCsvFilename(ret),
    );
    toast.success(t("Worksheet exported"), {
      description: t("Every line, each fuel type's subtotal and the grand total, as the form reads."),
    });
  };

  return (
    <div className="flex flex-wrap items-center gap-2">
      {transitions.has("generate") ? (
        <Button size="sm" onClick={() => generate.mutate()} disabled={generate.isPending}>
          <PlayIcon className="size-3.5" />
          {generate.isPending ? t("Generating...") : t("Generate the return")}
        </Button>
      ) : null}

      {transitions.has("recompute") && ret ? (
        <Button
          variant="outline"
          size="sm"
          onClick={() => recompute.mutate({ id: ret.id, version: ret.version })}
          disabled={recompute.isPending}
          title={t("Rebuild every line from the miles, fuel and rates on file now.")}
        >
          <RefreshCwIcon className="size-3.5" />
          {recompute.isPending ? t("Recomputing...") : t("Recompute")}
        </Button>
      ) : null}

      {transitions.has("finalize") ? (
        <Button
          variant="outline"
          size="sm"
          onClick={() => setDialog("finalize")}
          disabled={finalizeBlocked}
          title={
            missingRates.length > 0
              ? `${missingRates.length} member ${pluralize("line", missingRates.length)} has no published rate, so the return cannot be finalized yet.`
              : "Recompute and lock the worksheet."
          }
        >
          <LockIcon className="size-3.5" />
          {t("Finalize…")}
        </Button>
      ) : null}

      {transitions.has("reopen") ? (
        <Button variant="outline" size="sm" onClick={() => setDialog("reopen")}>
          <LockOpenIcon className="size-3.5" />
          {t("Reopen…")}
        </Button>
      ) : null}

      {transitions.has("markFiled") ? (
        <Button variant="outline" size="sm" onClick={() => setDialog("markFiled")}>
          <SendIcon className="size-3.5" />
          {t("Mark filed…")}
        </Button>
      ) : null}

      {transitions.has("amend") ? (
        <Button variant="outline" size="sm" onClick={() => setDialog("amend")}>
          <FilePlus2Icon className="size-3.5" />
          {t("Amend…")}
        </Button>
      ) : null}

      {transitions.has("export") ? (
        <Button variant="outline" size="sm" onClick={onExport} disabled={!ret}>
          <DownloadIcon className="size-3.5" />
          {t("Export CSV")}
        </Button>
      ) : null}

      {transitions.has("delete") ? (
        <Button variant="outline" size="sm" onClick={() => setDialog("delete")}>
          <Trash2Icon className="size-3.5" />
          {t("Delete draft…")}
        </Button>
      ) : null}

      {ret ? (
        <>
          <FinalizeReturnDialog
            open={dialog === "finalize"}
            onOpenChange={(open) => setDialog(open ? "finalize" : null)}
            ret={ret}
            period={period}
          />
          <ReopenReturnDialog
            open={dialog === "reopen"}
            onOpenChange={(open) => setDialog(open ? "reopen" : null)}
            ret={ret}
            period={period}
          />
          <MarkFiledDialog
            open={dialog === "markFiled"}
            onOpenChange={(open) => setDialog(open ? "markFiled" : null)}
            ret={ret}
            period={period}
          />
          <AmendReturnDialog
            open={dialog === "amend"}
            onOpenChange={(open) => setDialog(open ? "amend" : null)}
            ret={ret}
            period={period}
          />
          <DeleteReturnDialog
            open={dialog === "delete"}
            onOpenChange={(open) => setDialog(open ? "delete" : null)}
            ret={ret}
            period={period}
          />
        </>
      ) : null}
    </div>
  );
}
