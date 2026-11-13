import { BetaTag } from "@/components/beta-tag";
import { AttentionCountBadge } from "@/components/navigation/sidebar-chrome";
import type { ModuleAttention } from "@/components/navigation/sidebar-model";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { SectionPanel } from "@/components/section-panel";
import { useSidebarNavigation } from "@/hooks/use-sidebar-navigation";
import { findNavigationSection } from "@/lib/navigation-sections";
import { sectionIndexPanels, type SectionIndexPanel } from "@/lib/section-index";
import { ChevronRightIcon } from "@trenova/shared/components/icons";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { NotFoundRoute } from "@trenova/shared/components/error-boundary";
import { useT } from "@trenova/shared/i18n/use-t";
import { Link, useLocation } from "react-router";

function PanelRows({
  panel,
  attention,
}: {
  panel: SectionIndexPanel;
  attention: ReadonlyMap<string, ModuleAttention>;
}) {
  const t = useT();

  return (
    <ul className="divide-border flex flex-col divide-y">
      {panel.items.map((item) => (
        <li key={item.id}>
          <Link
            to={item.path}
            className="ui-focus-ring hover:bg-surface-hover group flex min-h-10 items-center gap-2 px-3 py-2 text-sm transition-colors outline-none"
          >
            <span className="min-w-0 truncate">{t(item.label)}</span>
            {item.includeBetaTag && <BetaTag />}
            <span className="ml-auto flex shrink-0 items-center gap-2">
              <AttentionCountBadge attention={attention.get(item.path)} />
              <ChevronRightIcon
                aria-hidden
                className="text-foreground-subtle group-hover:text-foreground size-3.5 transition-colors"
              />
            </span>
          </Link>
        </li>
      ))}
    </ul>
  );
}

/**
 * The page behind a crumb that names a part of the app rather than a page in
 * it: "Dispatch management", "Configuration files". It lists the pages the
 * person may open there, grouped the way the sidebar groups them, so climbing
 * the trail always lands somewhere useful.
 */
export function SectionIndexPage() {
  const t = useT();
  const { pathname } = useLocation();
  const section = findNavigationSection(pathname);
  const { views, pageAttention } = useSidebarNavigation();

  if (!section) {
    return <NotFoundRoute />;
  }

  const view = views.get(section.moduleId);
  const panels = view ? sectionIndexPanels(view, section.groupId) : [];
  const label = t(section.label);
  const panelTitle = (panel: SectionIndexPanel) => {
    if (panel.configuration) return t("Configuration");
    return panel.label ? t(panel.label) : t("Pages");
  };
  const moduleDescription = section.groupId === null ? view?.module.description : undefined;

  return (
    <PageLayout
      pageHeaderProps={{
        title: label,
        description: moduleDescription
          ? t(moduleDescription)
          : t("Every page in {0} you can open.", label),
      }}
    >
      {panels.length === 0 ? (
        <EmptySheet
          title={t("No pages you can open here")}
          description={t(
            "None of the pages here are open to you. An administrator can grant access or turn this part of Trenova on.",
          )}
          sketch={
            <div className="flex flex-col gap-2 rounded-lg border p-3">
              <GhostLine className="w-1/3" />
              <GhostLine className="w-2/3" />
              <GhostLine className="w-1/2" />
            </div>
          }
        />
      ) : (
        <div className="grid items-start gap-4 lg:grid-cols-2">
          {panels.map((panel) => (
            <SectionPanel key={panel.id} title={panelTitle(panel)} count={panel.items.length}>
              <PanelRows panel={panel} attention={pageAttention} />
            </SectionPanel>
          ))}
        </div>
      )}
    </PageLayout>
  );
}
