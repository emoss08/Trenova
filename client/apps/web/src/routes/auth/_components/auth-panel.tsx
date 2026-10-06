import { useT } from "@trenova/shared/i18n/use-t";
import logoRainbow from "@/assets/logo.webp";
import { edition } from "@/lib/edition";
import { updateService } from "@/services/update";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { useReducedMotion } from "motion/react";
import { AUTH_PITCH } from "./auth-ambient";

const AuthAmbient = edition.slots.AuthAmbient;

export type ReceiptRow = {
  key: string;
  value?: string;
};

export type CredentialReceipt = {
  rows: ReceiptRow[];
  issued: boolean;
};

/**
 * The ambient half of the sign-in screen. It is decoration with one exception: the
 * credential receipt, which fills in row by row as the real flow resolves identity,
 * workspace, roles and session — so the user can see what the session is being scoped
 * to before it is issued.
 *
 * Hidden below 900px, where the card takes the whole viewport.
 */
export function AuthPanel({ receipt }: { receipt: CredentialReceipt }) {
  const t = useT();

  const prefersReducedMotion = useReducedMotion();

  return (
    <aside className="border-border-2 bg-panel relative hidden min-w-0 flex-col justify-between overflow-hidden border-r p-10 min-[900px]:flex">
      <div className="auth-weave" />
      <div className="auth-aura" />
      {!prefersReducedMotion && <div className="auth-scan" />}

      <div className="relative flex items-center gap-2.5">
        <img src={logoRainbow} alt="" className="size-6 object-contain" />
        <span className="text-lg font-semibold tracking-[-0.02em]">{t("Trenova")}</span>
        <span className="border-border text-subtle-foreground font-table ml-0.5 border-l pl-2.5 text-2xs">
          {t("Enterprise")}
        </span>
      </div>

      <div className="relative flex flex-col gap-6">
        <h2 className="m-0 max-w-[19ch] text-4xl leading-[1.14] font-medium tracking-[-0.038em] text-balance">
          {AUTH_PITCH}
        </h2>
        <AuthAmbient />
        <CredentialReceiptCard receipt={receipt} />
      </div>

      <PanelFooter />
    </aside>
  );
}

function CredentialReceiptCard({ receipt }: { receipt: CredentialReceipt }) {
  const t = useT();

  return (
    <div className="auth-receipt border-border-2 relative w-full max-w-[392px] rounded-xl border">
      <div className="border-border-2 flex items-center justify-between border-b border-dashed px-3.5 py-[11px]">
        <span className="text-subtle-foreground font-table text-2xs whitespace-nowrap">
          {t("Credential")}
        </span>
        <span className="text-subtle-foreground font-table text-2xs whitespace-nowrap">
          {receipt.issued ? t("Issued") : t("Assembling")}
        </span>
      </div>
      {receipt.rows.map((row, index) => (
        <div
          key={row.key}
          className={cn(
            "border-border-2 grid grid-cols-[78px_1fr] items-baseline gap-3 border-b border-dashed px-3.5 py-2.5 last:border-b-0",
            // The stamp is pinned to the bottom-right corner and overlays the last row.
            receipt.issued && index === receipt.rows.length - 1 && "pr-[104px]",
          )}
        >
          <span className="text-subtle-foreground font-table text-2xs">{row.key}</span>
          {row.value ? (
            <span
              key={row.value}
              className="auth-receipt-fill min-w-0 truncate text-sm"
              title={row.value}
            >
              {row.value}
            </span>
          ) : (
            <span className="auth-receipt-pending" aria-hidden="true" />
          )}
        </div>
      ))}
      {receipt.issued && (
        <span className="auth-stamp border-foreground font-table absolute right-3.5 bottom-3 rounded-md border px-[7px] py-[3px] text-xs">
          {t("Authorized")}
        </span>
      )}
    </div>
  );
}

function PanelFooter() {
  const t = useT();

  // Public endpoint — it is the same call the update banner uses, and its success is
  // also the honest answer to whether the API is reachable from this browser.
  const versionQuery = useQuery({
    queryKey: ["system-version"],
    queryFn: updateService.getVersion,
    retry: false,
    staleTime: Number.POSITIVE_INFINITY,
  });
  const reachable = versionQuery.isSuccess;

  return (
    <div className="text-subtle-foreground font-table relative flex flex-wrap items-center gap-4 text-2xs whitespace-nowrap">
      <span className="inline-flex items-center gap-1.5">
        <i
          className={cn(
            "size-[5px] rounded-full",
            reachable ? "bg-foreground auth-pulse" : "bg-auth-danger",
          )}
        />
        {versionQuery.isPending
          ? t("Contacting network")
          : reachable
            ? t("Network operational")
            : t("Network unreachable")}
      </span>
      {versionQuery.data?.environment && <span>{versionQuery.data.environment}</span>}
      {versionQuery.data?.version && <span>{t("v{0}", versionQuery.data.version)}</span>}
    </div>
  );
}
