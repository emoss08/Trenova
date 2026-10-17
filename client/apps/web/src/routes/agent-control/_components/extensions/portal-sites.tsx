import { useApiMutation } from "@/hooks/use-api-mutation";
import { agentExtension } from "@/lib/queries/agent-extension";
import { apiService } from "@/services/api";
import type { PortalSite } from "@/types/agent-extension";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Edit05Icon, PlusIcon, Trash01Icon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { useState } from "react";
import { ConfirmDialog } from "../kit/modal";
import { PortalSiteDialog } from "./portal-site-dialog";
import { PortalSitesSkeleton } from "./portal-sites-skeleton";

type PortalSitesProps = {
  canUpdate: boolean;
};

/**
 * The portals agents may use. Agents reach nothing else: a site's addresses are the
 * browser's whole world for a task on it.
 */
export function PortalSites({ canUpdate }: PortalSitesProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const sites = useQuery(agentExtension.portalSites());
  const [editing, setEditing] = useState<PortalSite | null>(null);
  const [adding, setAdding] = useState(false);
  const [removing, setRemoving] = useState<PortalSite | null>(null);

  const remove = useApiMutation({
    resourceName: t("Portal site"),
    mutationFn: (site: PortalSite) => apiService.agentExtensionService.deletePortalSite(site.id),
    onSuccess: async () => {
      setRemoving(null);
      await queryClient.invalidateQueries({ queryKey: agentExtension._def });
    },
  });

  if (sites.isPending) {
    return <PortalSitesSkeleton />;
  }

  const items = sites.data?.items ?? [];

  return (
    <section aria-label={t("Portal sites")} className="flex flex-col gap-3">
      <div className="flex items-center justify-between gap-2">
        <h4 className="text-sm font-semibold">{t("Portal sites")}</h4>
        {canUpdate && (
          <Button type="button" size="sm" variant="outline" onClick={() => setAdding(true)}>
            <PlusIcon className="size-3.5" />
            {t("New portal site")}
          </Button>
        )}
      </div>
      {sites.isError ? (
        <Alert size="sm" variant="destructive">
          <AlertDescription>{t("The portal sites could not be loaded.")}</AlertDescription>
        </Alert>
      ) : items.length === 0 ? (
        <p className="text-muted-foreground text-sm">
          {t(
            "No portals yet. Add the portals your team signs in to, such as a warehouse's dock appointment site, and agents can look things up there.",
          )}
        </p>
      ) : (
        <ul className="divide-border flex flex-col divide-y rounded-md border">
          {items.map((site) => (
            <li key={site.id} className="flex items-start gap-3 px-3 py-2.5">
              <div className="flex min-w-0 flex-1 flex-col gap-1">
                <div className="flex flex-wrap items-center gap-1.5">
                  <span className="truncate text-sm font-medium">{site.name}</span>
                  {!site.enabled && <Badge variant="neutral">{t("Off")}</Badge>}
                  {site.allowActions ? (
                    <Badge variant="info">{t("Agents may act")}</Badge>
                  ) : (
                    <Badge variant="neutral">{t("Read only")}</Badge>
                  )}
                  {site.hasSignIn && <Badge variant="neutral">{t("Signs in")}</Badge>}
                </div>
                {site.purpose && (
                  <p className="text-muted-foreground text-xs">{site.purpose}</p>
                )}
                <p className="text-muted-foreground truncate font-mono text-xs">
                  {site.domains.join(", ")}
                </p>
              </div>
              {canUpdate && (
                <div className="flex shrink-0 items-center gap-1">
                  <Button
                    type="button"
                    size="icon-sm"
                    variant="ghost"
                    aria-label={t("Edit {0}", site.name)}
                    onClick={() => setEditing(site)}
                  >
                    <Edit05Icon className="size-3.5" />
                  </Button>
                  <Button
                    type="button"
                    size="icon-sm"
                    variant="ghost"
                    aria-label={t("Remove {0}", site.name)}
                    onClick={() => setRemoving(site)}
                  >
                    <Trash01Icon className="size-3.5" />
                  </Button>
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
      <PortalSiteDialog
        open={adding || editing !== null}
        onOpenChange={(open) => {
          if (!open) {
            setAdding(false);
            setEditing(null);
          }
        }}
        site={editing}
      />
      <ConfirmDialog
        open={removing !== null}
        onClose={() => setRemoving(null)}
        title={removing ? t("Remove {0}?", removing.name) : ""}
        description={t(
          "Agents stop using it at once. A site agents have used keeps its history; turn it off instead.",
        )}
        confirmLabel={t("Remove site")}
        danger
        busy={remove.isPending}
        onConfirm={() => removing && remove.mutate(removing)}
      />
    </section>
  );
}
