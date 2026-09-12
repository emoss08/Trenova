import { useT } from "@trenova/shared/i18n/use-t";
import type { WorkerCredentialRow } from "@/lib/graphql/worker-credential";
import { Button } from "@trenova/shared/components/ui/button";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { ChevronDownIcon } from "lucide-react";
import { useState } from "react";

/**
 * Archived credentials, folded away. They stay because an auditor asks what
 * was held and when, not because anybody reads them day to day.
 */
export function CredentialHistory({ archived }: { archived: readonly WorkerCredentialRow[] }) {
  const t = useT();

  const [open, setOpen] = useState(false);
  if (archived.length === 0) return null;

  return (
    <section className="flex flex-col gap-2">
      <Button
        variant="ghost"
        size="sm"
        className="text-muted-foreground w-fit px-1"
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
      >
        <ChevronDownIcon className={cn("size-3.5 transition-transform", open && "rotate-180")} />
        {t("History ({0})", archived.length)}
      </Button>
      {open ? (
        <ul className="divide-border divide-y rounded-lg border">
          {archived.map((credential) => (
            <li
              key={credential.id}
              data-testid={`credential-history-${credential.id}`}
              className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-4 px-3 py-2.5 text-xs"
            >
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">
                  {credential.credentialType?.name ?? t("Credential")}
                </p>
                <p className="text-muted-foreground truncate">
                  {[
                    credential.number,
                    credential.expiresAt
                      ? `Expired ${formatUnixDateMedium(credential.expiresAt)}`
                      : "No expiry",
                    credential.archivedAt
                      ? `Archived ${formatUnixDateMedium(credential.archivedAt)}`
                      : null,
                  ]
                    .filter(Boolean)
                    .join(" · ")}
                </p>
              </div>
              {credential.archiveReason ? (
                <span className="text-muted-foreground text-right">{credential.archiveReason}</span>
              ) : null}
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  );
}
