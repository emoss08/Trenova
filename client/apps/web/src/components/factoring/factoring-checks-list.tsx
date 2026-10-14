import { recordPath } from "@/config/record-links";
import { usePermission } from "@/hooks/use-permission";
import { factoringFixActionLabel } from "@/lib/factoring";
import type { FactoringCheck, FactoringInvoiceState } from "@/lib/graphql/factoring";
import {
  AlertCircleIcon,
  CheckCircleIcon,
  LinkExternal01Icon,
  Mail01Icon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { Link } from "react-router";
import { RequestFactoringDocumentsDialog } from "./request-factoring-documents-dialog";

type CheckLink = { label: string; to: string };

/**
 * The page a fix is made on. A missing agreement or notice is set up on the
 * factor, a dispute on the invoice's disputes tab, and a shipment problem on
 * the shipment; a missing document is asked for in place instead.
 */
function checkLinks(
  check: FactoringCheck,
  state: Pick<FactoringInvoiceState, "invoiceId" | "factoringCompanyId" | "customerId">,
  t: ReturnType<typeof useT>,
): CheckLink[] {
  switch (check.fix) {
    case "SetUpAgreement":
      return state.factoringCompanyId
        ? [
            {
              label: t("Open the factor's agreements"),
              to: recordPath("factoring_company", state.factoringCompanyId),
            },
          ]
        : [];
    case "SendNoa":
    case "RecordNoaAcknowledgement":
      return [
        ...(state.factoringCompanyId
          ? [
              {
                label: t("Open the factor's notices"),
                to: recordPath("factoring_company", state.factoringCompanyId),
              },
            ]
          : []),
        { label: t("Open the customer"), to: recordPath("customer", state.customerId) },
      ];
    case "ResolveDispute":
      return [
        {
          label: t("Open the disputes"),
          to: recordPath("invoice", state.invoiceId, { tab: "disputes" }),
        },
      ];
    case "ReviewShipment":
      return check.shipmentId
        ? [{ label: t("Open the shipment"), to: recordPath("shipment", check.shipmentId) }]
        : [];
    default:
      return [];
  }
}

/** Failed checks first, in the order the server ran them; passed checks after. */
export function orderedChecks(checks: readonly FactoringCheck[]): FactoringCheck[] {
  return [...checks.filter((check) => !check.passed), ...checks.filter((check) => check.passed)];
}

export function FactoringChecksList({
  state,
  invoiceNumber,
  showPassed = true,
}: {
  state: FactoringInvoiceState;
  invoiceNumber: string;
  showPassed?: boolean;
}) {
  const t = useT();
  const [requesting, setRequesting] = useState<FactoringCheck | null>(null);
  const { allowed: canRequest } = usePermission(Resource.CustomerCommunication, Operation.Create);
  const checks = orderedChecks(state.checks).filter((check) => showPassed || !check.passed);

  if (checks.length === 0) {
    return <p className="text-muted-foreground px-3 py-2 text-sm">{t("Every check passed.")}</p>;
  }

  return (
    <>
      <ul className="divide-y" data-testid="factoring-checks">
        {checks.map((check, index) => {
          const links = check.passed ? [] : checkLinks(check, state, t);
          return (
            <li
              key={`${check.code}:${check.documentTypeId ?? ""}:${check.shipmentId ?? ""}:${index}`}
              className="flex items-start gap-2.5 px-3 py-2 text-sm"
            >
              {check.passed ? (
                <CheckCircleIcon className="text-success-foreground mt-0.5 size-4 shrink-0" />
              ) : (
                <AlertCircleIcon className="text-warning-foreground mt-0.5 size-4 shrink-0" />
              )}
              <div className="min-w-0 flex-1 space-y-1">
                <p className={cn(check.passed && "text-muted-foreground")}>{check.message}</p>
                {check.detail ? (
                  <p className="text-muted-foreground text-xs">{check.detail}</p>
                ) : null}
                {!check.passed && check.fix ? (
                  <div className="flex flex-wrap items-center gap-2 pt-0.5">
                    {check.fix === "RequestDocument" && canRequest ? (
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={() => setRequesting(check)}
                      >
                        <Mail01Icon />
                        {factoringFixActionLabel(check.fix)}
                      </Button>
                    ) : null}
                    {links.map((link) => (
                      <Link
                        key={link.to}
                        to={link.to}
                        className="text-brand inline-flex items-center gap-1 text-xs font-medium hover:underline"
                      >
                        {link.label}
                        <LinkExternal01Icon className="size-3" />
                      </Link>
                    ))}
                    {check.fix !== "RequestDocument" && links.length === 0 ? (
                      <span className="text-muted-foreground text-xs">
                        {factoringFixActionLabel(check.fix)}
                      </span>
                    ) : null}
                  </div>
                ) : null}
              </div>
            </li>
          );
        })}
      </ul>
      <RequestFactoringDocumentsDialog
        invoiceId={state.invoiceId}
        invoiceNumber={invoiceNumber}
        documentTypeIds={requesting?.documentTypeId ? [requesting.documentTypeId] : undefined}
        documentNames={requesting?.documentTypeName ? [requesting.documentTypeName] : []}
        open={requesting !== null}
        onOpenChange={(open) => {
          if (!open) setRequesting(null);
        }}
      />
    </>
  );
}
