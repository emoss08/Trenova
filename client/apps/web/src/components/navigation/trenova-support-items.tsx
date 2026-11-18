import { newSupportRequestUrl, supportRequestPage } from "@/lib/trenova-support";
import { usePublicConfig } from "@trenova/shared/hooks/use-public-config";
import { useT } from "@trenova/shared/i18n/use-t";
import { Inbox01Icon, LifeBuoy01Icon, LinkExternal01Icon } from "@trenova/shared/components/icons";
import { DropdownMenuLinkItem } from "@trenova/shared/components/ui/dropdown-menu";
import { useRouterState } from "@tanstack/react-router";

/**
 * On a Managed Trenova instance, the way to Trenova's support from the account menu: a new
 * request in the console, with this instance chosen and the page the person was on, and the
 * organization's requests. Each opens the console in a new tab. Nothing on a self-hosted
 * instance, or when the control plane sent no address.
 */
export function TrenovaSupportItems() {
  const t = useT();
  const { config } = usePublicConfig();
  const routePath = useRouterState({ select: (state) => state.matches.at(-1)?.fullPath });

  if (!config.managed) {
    return null;
  }
  const { requestsUrl, newRequestUrl } = config.trenovaSupport;
  const external = <LinkExternal01Icon className="text-muted-foreground size-3.5" />;

  return (
    <>
      {newRequestUrl && (
        <DropdownMenuLinkItem
          href={newSupportRequestUrl(newRequestUrl, supportRequestPage(routePath))}
          target="_blank"
          rel="noopener noreferrer"
          title={t("Contact Trenova support")}
          startContent={<LifeBuoy01Icon className="size-4" />}
          endContent={external}
        />
      )}
      {requestsUrl && (
        <DropdownMenuLinkItem
          href={requestsUrl}
          target="_blank"
          rel="noopener noreferrer"
          title={t("Your support requests")}
          startContent={<Inbox01Icon className="size-4" />}
          endContent={external}
        />
      )}
    </>
  );
}
