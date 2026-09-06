import type { WorkerCredentialRow } from "@/lib/graphql/worker-credential";
import { Button } from "@trenova/shared/components/ui/button";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { ChevronDownIcon } from "lucide-react";
import { useState } from "react";

export function CredentialHistory({ archived }: { archived: readonly WorkerCredentialRow[] }) {
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
        <ChevronDownIcon className={`size-3.5 transition-transform ${open ? "rotate-180" : ""}`} />
        History ({archived.length})
      </Button>
      {open ? (
        <ul className="divide-border border-border divide-y rounded-lg border">
          {archived.map((credential) => (
            <li
              key={credential.id}
              data-testid={`credential-history-${credential.id}`}
              className="flex flex-wrap items-center justify-between gap-2 px-3 py-2 text-xs"
            >
              <div className="min-w-0">
                <p className="truncate font-medium">
                  {credential.credentialType?.name ?? "Credential"}
                  {credential.number ? (
                    <span className="text-muted-foreground ml-1 font-normal tabular-nums">
                      {credential.number}
                    </span>
                  ) : null}
                </p>
                <p className="text-muted-foreground">
                  {credential.expiresAt
                    ? `Expired ${formatUnixDateMedium(credential.expiresAt)}`
                    : "No expiry"}
                  {credential.archivedAt
                    ? ` · Archived ${formatUnixDateMedium(credential.archivedAt)}`
                    : ""}
                </p>
              </div>
              {credential.archiveReason ? (
                <span className="text-muted-foreground italic">{credential.archiveReason}</span>
              ) : null}
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  );
}
