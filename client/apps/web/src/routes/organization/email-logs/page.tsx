import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { formatUnixDateTime } from "@trenova/shared/lib/date";

export function EmailLogsPage() {
  const logsQuery = useQuery(queries.email.logs());
  const logs = logsQuery.data?.results ?? [];

  return (
    <div className="flex h-full flex-col gap-4 p-4">
      <div>
        <h1 className="text-lg font-semibold">Email Logs</h1>
        <p className="text-muted-foreground text-sm">Transactional email send and delivery history.</p>
      </div>
      <section className="overflow-hidden rounded-md border">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-muted-foreground border-b text-left text-xs uppercase">
            <tr>
              <th className="px-3 py-2">Subject</th>
              <th className="px-3 py-2">Purpose</th>
              <th className="px-3 py-2">Recipients</th>
              <th className="px-3 py-2">Status</th>
              <th className="px-3 py-2">Attempts</th>
              <th className="px-3 py-2">Created</th>
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
                  <span className="rounded border px-2 py-1 text-xs">{log.status}</span>
                </td>
                <td className="px-3 py-2">{log.attempts}</td>
                <td className="px-3 py-2">{formatUnixDateTime(log.createdAt)}</td>
              </tr>
            ))}
            {logs.length === 0 && (
              <tr>
                <td className="text-muted-foreground px-3 py-8 text-center" colSpan={6}>
                  No email logs yet.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </section>
    </div>
  );
}
