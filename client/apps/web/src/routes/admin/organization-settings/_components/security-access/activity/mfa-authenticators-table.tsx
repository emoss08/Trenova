import { DataTable } from "@/components/data-table/data-table";
import { mfaAuthenticatorTableGraphQLConfig, type MFAAuthenticatorRow } from "@/lib/graphql/security-activity-table";
import { useT } from "@trenova/shared/i18n/use-t";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { mfaAuthenticatorColumns } from "./activity-columns";

export default function MFAAuthenticatorsTable() {
  const t = useT();
  const columns = useMemo(() => mfaAuthenticatorColumns(t), [t]);

  return (
    <DataTable<MFAAuthenticatorRow>
      name="MFA Authenticator"
      emptyTitle={t("No two-factor devices enrolled yet")}
      queryKey="security-mfa-authenticators"
      graphql={mfaAuthenticatorTableGraphQLConfig}
      resource={Resource.MFAAuthenticator}
      columns={columns}
      enableCreateAction={false}
    />
  );
}
