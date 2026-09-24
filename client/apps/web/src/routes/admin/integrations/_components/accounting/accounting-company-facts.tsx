import type { AccountingConnection } from "@/lib/graphql/accounting-sync";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";

export function AccountingCompanyFacts({ connection }: { connection: AccountingConnection }) {
  const t = useT();

  return (
    <DescriptionList columns={2}>
      <DescriptionItem label={t("Company")}>
        {connection.externalCompanyName || <DescriptionEmpty />}
      </DescriptionItem>
      <DescriptionItem label={t("Legal name")}>
        {connection.externalLegalName || <DescriptionEmpty />}
      </DescriptionItem>
      <DescriptionItem label={t("Country")}>
        {connection.externalCountry || <DescriptionEmpty />}
      </DescriptionItem>
      <DescriptionItem label={t("Home currency")}>
        {connection.externalHomeCurrency || <DescriptionEmpty />}
      </DescriptionItem>
      <DescriptionItem label={t("Multicurrency")}>
        {connection.externalMultiCurrencyEnabled ? t("On") : t("Off")}
      </DescriptionItem>
      <DescriptionItem label={t("Books closed through")} numeric>
        {connection.externalBooksClosedThrough
          ? formatUnixDateMedium(connection.externalBooksClosedThrough, { timezone: "UTC" })
          : t("No closing date")}
      </DescriptionItem>
    </DescriptionList>
  );
}
