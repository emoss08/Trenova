import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { CheckIcon, ClipboardIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { scimBaseUrl } from "./constants";

export function SCIMBaseURL({ tenantSlug }: { tenantSlug: string }) {
  const t = useT();
  const { copy, isCopied } = useCopyToClipboard();
  const url = scimBaseUrl(tenantSlug);

  return (
    <div className="mt-3 flex flex-col gap-1">
      <span className="text-muted-foreground text-xs font-medium">{t("SCIM base URL")}</span>
      <div className="flex items-center gap-2">
        <code className="bg-card text-foreground min-w-0 flex-1 truncate rounded-md border px-2 py-1 font-mono text-xs">
          {url}
        </code>
        <Button size="sm" variant="outline" onClick={() => void copy(url, { withToast: true })}>
          {isCopied ? <CheckIcon /> : <ClipboardIcon />}
          {isCopied ? t("Copied") : t("Copy")}
        </Button>
      </div>
    </div>
  );
}
