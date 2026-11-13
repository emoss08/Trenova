import { BetaTag } from "@/components/beta-tag";
import { HoverGlideGroup } from "@/components/hover-glide";
import { AttentionCountBadge } from "@/components/navigation/sidebar-chrome";
import type { ModuleAttention, SidebarPageItem } from "@/components/navigation/sidebar-model";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { useSidebarNavigation } from "@/hooks/use-sidebar-navigation";
import { findNavigationSection } from "@/lib/navigation-sections";
import { sectionIndexPanels, type SectionIndexPanel } from "@/lib/section-index";
import { ArrowRightIcon } from "@trenova/shared/components/icons";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { NotFoundRoute } from "@trenova/shared/components/error-boundary";
import { useT } from "@trenova/shared/i18n/use-t";
import { useId } from "react";
import { Link, useLocation } from "react-router";
import { PAGE_DESCRIPTIONS } from "./page-descriptions.gen";

const descriptions: Readonly<Record<string, string>> = PAGE_DESCRIPTIONS;

/**
 * One page as a card to choose: its name, how much work waits there, and what it
 * is for in the page's own words. The card is flat and draws no hover fill of its
 * own; the group's glide slides under whichever card the pointer is on.
 */
function PageCard({
  item,
  attention,
}: {
  item: SidebarPageItem;
  attention: ModuleAttention | undefined;
}) {
  const t = useT();
  const description = descriptions[item.path];

  return (
    <Link
      to={item.path}
      data-glide-target
      className="group ui-focus-ring relative flex min-h-22 flex-col gap-1 rounded-lg border p-3 outline-none"
    >
      <span className="flex items-center gap-2">
        <span className="min-w-0 truncate text-sm font-medium">{t(item.label)}</span>
        {item.includeBetaTag && <BetaTag />}
        <span className="ml-auto flex shrink-0 items-center gap-2">
          <AttentionCountBadge attention={attention} />
          <ArrowRightIcon
            aria-hidden
            className="size-3.5 text-foreground-subtle transition-colors group-hover:text-foreground"
          />
        </span>
      </span>
      {description ? (
        <span className="line-clamp-2 text-xs leading-relaxed text-muted-foreground">
          {description}
        </span>
      ) : null}
    </Link>
  );
}

function PanelGroup({
  panel,
  title,
  attention,
}: {
  panel: SectionIndexPanel;
  title: string | null;
  attention: ReadonlyMap<string, ModuleAttention>;
}) {
  const headingId = useId();

  return (
    <section aria-labelledby={title ? headingId : undefined} className="flex flex-col gap-2">
      {title ? (
        <h2 id={headingId} className="flex items-baseline gap-2 text-sm font-semibold">
          {title}
          <span className="text-xs font-normal text-muted-foreground tabular-nums">
            {panel.items.length}
          </span>
        </h2>
      ) : null}
      <HoverGlideGroup
        glideClassName="rounded-lg"
        className="grid gap-2 sm:grid-cols-2 xl:grid-cols-3"
      >
        {panel.items.map((item) => (
          <PageCard key={item.id} item={item} attention={attention.get(item.path)} />
        ))}
      </HoverGlideGroup>
    </section>
  );
}

function SectionIndexSketch() {
  return (
    <div className="grid grid-cols-3 gap-2">
      {[0, 1, 2].map((index) => (
        <div key={index} className="flex h-16 flex-col gap-2 rounded-lg border p-3">
          <GhostLine className="w-1/2" />
          <GhostLine className="w-5/6" />
        </div>
      ))}
    </div>
  );
}

/**
 * The page behind a crumb that names a part of the app rather than a page in
 * it: "Dispatch management", "Configuration files". It lays out the pages the
 * person may open there as cards that say what each is for, grouped the way the
 * sidebar groups them, so climbing the trail lands on a directory to choose
 * from rather than a dead end.
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
  const moduleDescription = section.groupId === null ? view?.module.description : undefined;
  const panelTitle = (panel: SectionIndexPanel) => {
    if (panel.configuration) return t("Configuration");
    return panel.label ? t(panel.label) : null;
  };

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
          sketch={<SectionIndexSketch />}
        />
      ) : (
        <div className="flex flex-col gap-6">
          {panels.map((panel) => (
            <PanelGroup
              key={panel.id}
              panel={panel}
              title={panelTitle(panel)}
              attention={pageAttention}
            />
          ))}
        </div>
      )}
    </PageLayout>
  );
}
