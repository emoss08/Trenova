import { useT } from "@trenova/shared/i18n/use-t";
import { SidebarSectionLabel } from "@/components/navigation/sidebar-primitives";
import { useSidebarQuickActions } from "@/hooks/use-sidebar-quick-actions";
import { Link } from "react-router";

export function QuickActionsSection() {
  const t = useT();

  const actions = useSidebarQuickActions();

  if (actions.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-1">
      <SidebarSectionLabel>{t("Quick Actions")}</SidebarSectionLabel>
      <div className="grid grid-cols-2 gap-1.5 px-0.5">
        {actions.map(({ definition, icon: Icon, href, shortLabel }) => (
          <Link
            key={definition.id}
            to={href}
            title={t(definition.description)}
            className="border-border bg-background text-foreground/80 hover:bg-muted hover:text-foreground flex h-7 items-center gap-1.5 truncate rounded-md border px-2 text-xs font-medium transition-colors"
          >
            <Icon className="text-muted-foreground size-3.5 shrink-0" strokeWidth={1.75} />
            <span className="truncate">{shortLabel}</span>
          </Link>
        ))}
      </div>
    </div>
  );
}
