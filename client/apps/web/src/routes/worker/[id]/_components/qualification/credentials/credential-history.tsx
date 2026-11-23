import type { WorkerCredentialRow } from "@/lib/graphql/worker-credential";
import { ChevronRightIcon } from "@trenova/shared/components/icons";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useId, useState } from "react";

/**
 * Archived credentials and renewals, folded under the table. They stay because an auditor asks
 * what was held and when, not because anybody reads them day to day.
 */
export function CredentialHistory({ archived }: { archived: readonly WorkerCredentialRow[] }) {
  const t = useT();
  const listId = useId();
  const [open, setOpen] = useState(false);
  if (archived.length === 0) return null;

  return (
    <div className="border-t">
      <button
        type="button"
        aria-expanded={open}
        aria-controls={listId}
        onClick={() => setOpen((value) => !value)}
        className="group flex h-9 w-full items-center gap-1.5 ui-focus-ring pr-3 pl-3.5 text-left text-xs font-medium"
      >
        <ChevronRightIcon
          aria-hidden
          className={cn(
            "size-3.5 text-foreground-subtle transition-transform duration-200 ease-settle",
            open && "rotate-90",
          )}
        />
        {t("History")}
        <span className="font-normal text-foreground-subtle tabular-nums">{archived.length}</span>
        <span className="ml-1 font-normal text-muted-foreground group-hover:text-foreground">
          {t("archived credentials and renewals")}
        </span>
      </button>
      {open ? (
        <ScrollArea className="max-h-72" maskVariant="card">
          <ul
            id={listId}
            className="animate-expand-in border-t border-border-subtle motion-reduce:animate-none"
          >
            {archived.map((credential) => (
              <li
                key={credential.id}
                data-testid={`credential-history-${credential.id}`}
                className="grid min-h-9 grid-cols-[minmax(0,1.3fr)_minmax(0,1.2fr)_minmax(0,1fr)_minmax(0,1fr)] items-center gap-3 py-2 pr-3 pl-4 text-xs not-first:border-t not-first:border-border-subtle"
              >
                <span className="truncate text-sm font-medium text-muted-foreground">
                  {credential.credentialType?.name ?? t("Credential")}
                </span>
                <span className="truncate font-mono text-muted-foreground">
                  {credential.issuedAt && credential.expiresAt
                    ? t(
                        "{0} to {1}",
                        formatUnixDateMedium(credential.issuedAt),
                        formatUnixDateMedium(credential.expiresAt),
                      )
                    : credential.expiresAt
                      ? t("Expired {0}", formatUnixDateMedium(credential.expiresAt))
                      : t("No expiry")}
                </span>
                <span className="truncate font-mono text-muted-foreground">
                  {credential.archivedAt
                    ? t("Archived {0}", formatUnixDateMedium(credential.archivedAt))
                    : null}
                </span>
                <span className="truncate text-muted-foreground">{credential.archiveReason}</span>
              </li>
            ))}
          </ul>
        </ScrollArea>
      ) : null}
    </div>
  );
}
