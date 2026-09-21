import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { formatUnixDateTime } from "@trenova/shared/lib/date";

export function EmailLogsPage() {
  const t = useT();

  const logsQuery = useQuery(queries.email.logs());
  const logs = logsQuery.data?.results ?? [];

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Email Logs"),
        description: t("Transactional email send and delivery history."),
      }}
    >
      <section className="bg-card overflow-hidden rounded-lg border">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-muted-foreground border-b text-left text-xs">
            <tr>
              <th className="px-3 py-2">{t("Subject")}</th>
              <th className="px-3 py-2">{t("Purpose")}</th>
              <th className="px-3 py-2">{t("Recipients")}</th>
              <th className="px-3 py-2">{t("Status")}</th>
              <th className="px-3 py-2">{t("Attempts")}</th>
              <th className="px-3 py-2">{t("Created")}</th>
            </tr>
          </thead>
          <tbody>
            {logs.map((log) => (
              <tr key={log.id} className="border-b last:border-0">
                <td className="px-3 py-2 font-medium">
                  <div>{log.subject}</div>
                  {log.lastError && <div className="text-destructive text-xs">{log.lastError}</div>}
                </td>
                <td className="px-3 py-2">{log.purpose}</td>
                <td className="px-3 py-2">{log.toRecipients.join(", ")}</td>
                <td className="px-3 py-2">
                  <span className="rounded-md border px-2 py-1 text-xs">{log.status}</span>
                </td>
                <td className="px-3 py-2">{log.attempts}</td>
                <td className="px-3 py-2">{formatUnixDateTime(log.createdAt)}</td>
              </tr>
            ))}
            {logs.length === 0 && (
              <tr>
                <td className="text-muted-foreground px-3 py-8 text-center" colSpan={6}>
                  {t("No email logs yet.")}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </section>
    </PageLayout>
  );
}
