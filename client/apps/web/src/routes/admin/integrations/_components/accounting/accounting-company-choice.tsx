import { ChoiceButton } from "@/components/choice-button";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { describeApiError } from "@/lib/api-error-message";
import {
  chooseAccountingCompany,
  type AccountingAuthorizationResult,
  type AccountingConnection,
} from "@/lib/graphql/accounting-sync";
import { useMutation } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { useState, type ReactNode } from "react";
import type { AccountingVendor } from "./accounting-vendors";

export type AccountingCompanyChoiceOffer = Extract<
  AccountingAuthorizationResult,
  { kind: "choose" }
>;

type AccountingCompanyChoiceProps = {
  vendor: AccountingVendor;
  choice: AccountingCompanyChoiceOffer;
  onConnected: (connection: AccountingConnection) => void;
  cancelAction: ReactNode;
  expired: ReactNode;
};

export function AccountingCompanyChoice({
  vendor,
  choice,
  onConnected,
  cancelAction,
  expired,
}: AccountingCompanyChoiceProps) {
  const t = useT();
  const nowSeconds = useNowSeconds(15_000);
  const [companyId, setCompanyId] = useState<string | null>(null);
  const choose = useMutation({
    mutationFn: (id: string) =>
      chooseAccountingCompany({
        integrationType: vendor.system,
        choiceToken: choice.choiceToken,
        companyId: id,
      }),
    onSuccess: onConnected,
  });

  const lapsed = choice.choiceExpiresAt !== null && nowSeconds >= choice.choiceExpiresAt;
  if (lapsed && !choose.isPending && !choose.isSuccess) {
    return expired;
  }

  return (
    <div className="space-y-4">
      <div className="space-y-1">
        <h2 className="text-base font-semibold">{t("Choose the organisation")}</h2>
        <p className="text-foreground-muted text-sm">
          {t(
            "{0} gave Trenova access to more than one organisation. Choose the one whose books this Trenova organization keeps; Trenova gives up its access to the others.",
            vendor.name,
          )}
        </p>
        {choice.choiceExpiresAt !== null ? (
          <p className="text-foreground-subtle text-xs">
            {t(
              "Choose by {0}. After that, start the connection again.",
              formatUnixTime(choice.choiceExpiresAt),
            )}
          </p>
        ) : null}
      </div>
      <div
        role="radiogroup"
        aria-label={t("Organisations in {0}", vendor.name)}
        className="flex flex-col gap-2"
      >
        {choice.companies.map((company) => (
          <ChoiceButton
            key={company.id}
            selected={companyId === company.id}
            onClick={() => setCompanyId(company.id)}
            disabled={choose.isPending || choose.isSuccess}
          >
            <span className="font-medium">{company.name || company.id}</span>
          </ChoiceButton>
        ))}
      </div>
      {choose.isError ? (
        <Alert size="sm" variant="destructive">
          <AlertDescription>{describeApiError(choose.error)}</AlertDescription>
        </Alert>
      ) : null}
      <div className="flex justify-end gap-2 border-t pt-4">
        {cancelAction}
        <Button
          type="button"
          disabled={companyId === null || choose.isSuccess}
          isLoading={choose.isPending}
          loadingText={t("Connecting...")}
          onClick={() => {
            if (companyId !== null) {
              choose.mutate(companyId);
            }
          }}
        >
          {t("Connect this organisation")}
        </Button>
      </div>
    </div>
  );
}
