import { useQuery } from "@tanstack/react-query";
import { Certificate01Icon, CheckIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import type { WorkerCredentialTypeChoice } from "@/lib/graphql/worker-credential";
import { worker } from "@/lib/queries/worker";

/**
 * Narrows the roster to the workers with no valid credential of one type:
 * none on file, archived or expired. The choice is a credential type's code,
 * which is what the roster's read takes.
 */
export function MissingCredentialMenu({
  value,
  onChange,
}: {
  value: string | null;
  onChange: (code: string | null) => void;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const choices = useQuery({ ...worker.credentialTypeChoices(), enabled: open || value !== null });
  const types = choices.data ?? [];
  const chosen = types.find((type) => type.code === value);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            variant="outline"
            size="sm"
            aria-label={t("Missing credential")}
            data-active={value !== null}
            className="data-[active=true]:bg-brand-subtle data-[active=true]:text-brand-subtle-foreground data-[active=true]:border-brand/30 shrink-0"
          >
            <Certificate01Icon className="size-3.5" />
            <span>{chosen ? t("Missing {0}", chosen.name) : t("Missing credential")}</span>
          </Button>
        }
      />
      <PopoverContent align="start" className="w-64 gap-1 p-1">
        <span className="text-muted-foreground px-2 pt-1 pb-0.5 text-xs font-medium">
          {t("Workers with none on file, archived or expired")}
        </span>
        {choices.isPending ? (
          <MissingCredentialMenuSkeleton />
        ) : choices.isError ? (
          <p className="text-muted-foreground px-2 py-1.5 text-sm">
            {t("Credential types could not be loaded.")}
          </p>
        ) : types.length === 0 ? (
          <p className="text-muted-foreground px-2 py-1.5 text-sm">
            {t("No credential types are tracked yet.")}
          </p>
        ) : (
          <div role="radiogroup" aria-label={t("Missing credential")} className="flex flex-col">
            {types.map((type) => (
              <CredentialChoice
                key={type.id}
                type={type}
                selected={type.code === value}
                onSelect={() => {
                  setOpen(false);
                  onChange(type.code === value ? null : type.code);
                }}
              />
            ))}
          </div>
        )}
      </PopoverContent>
    </Popover>
  );
}

function CredentialChoice({
  type,
  selected,
  onSelect,
}: {
  type: WorkerCredentialTypeChoice;
  selected: boolean;
  onSelect: () => void;
}) {
  const t = useT();
  return (
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      onClick={onSelect}
      className={cn(
        "ui-focus-ring hover:bg-surface-active flex h-7 cursor-pointer items-center gap-2 rounded-md px-2 text-left text-sm",
        selected ? "text-foreground" : "text-muted-foreground hover:text-foreground",
      )}
    >
      <span className="flex-1 truncate">{type.name}</span>
      {type.isRequired ? (
        <span className="text-muted-foreground text-xs">{t("Required")}</span>
      ) : null}
      {selected ? <CheckIcon className="size-3.5 shrink-0" /> : null}
    </button>
  );
}

function MissingCredentialMenuSkeleton() {
  const t = useT();
  return (
    <div
      aria-busy="true"
      aria-label={t("Loading credential types")}
      className="flex flex-col gap-1 p-1"
    >
      {[0, 1, 2, 3].map((row) => (
        <div key={row} className="ui-shimmer h-6 rounded-md" />
      ))}
    </div>
  );
}
