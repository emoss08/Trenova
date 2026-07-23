import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
} from "@/components/ui/sidebar";
import { SettingsIcon, UsersIcon, type LucideIcon } from "lucide-react";
import React, { Activity, useEffect, useState } from "react";
import { SamsaraConfigurationContent } from "./configuration-content";
import { SamsaraWorkerSyncCard } from "./sync/samsara-worker-sync-card";

type SamsaraView = "configuration" | "worker-sync";

type NavItem = {
  label: string;
  icon: LucideIcon;
  value: SamsaraView;
};

const navItems: NavItem[] = [
  {
    label: "Configuration",
    icon: SettingsIcon,
    value: "configuration",
  },
  {
    label: "Worker Sync",
    icon: UsersIcon,
    value: "worker-sync",
  },
];

export function SamsaraIntegrationModal({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [activeView, setActiveView] = useState<SamsaraView>("configuration");

  useEffect(() => {
    if (!open) {
      setActiveView("configuration");
    }
  }, [open]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="overflow-hidden p-0 md:max-h-[92vh] md:max-w-295 lg:max-w-330 xl:max-w-360">
        <DialogTitle className="sr-only">Settings</DialogTitle>
        <DialogDescription className="sr-only">Customize your settings here.</DialogDescription>
        <SidebarProvider>
          <Sidebar collapsible="none" className="hidden border-r border-border md:flex">
            <SidebarContent>
              <SidebarGroup>
                <SidebarGroupContent>
                  <SidebarMenu className="gap-1 p-2">
                    {navItems.map((item) => (
                      <SidebarMenuItem key={item.value} className="cursor-pointer">
                        <SidebarMenuButton
                          isActive={activeView === item.value}
                          onClick={() => setActiveView(item.value)}
                          className="cursor-pointer"
                        >
                          <item.icon className="size-4" />
                          <span>{item.label}</span>
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    ))}
                  </SidebarMenu>
                </SidebarGroupContent>
              </SidebarGroup>
            </SidebarContent>
          </Sidebar>
          <InnerContent>
            <Activity mode={activeView === "configuration" ? "visible" : "hidden"}>
              <SamsaraConfigurationContent open={activeView === "configuration"} />
            </Activity>
            <Activity mode={activeView === "worker-sync" ? "visible" : "hidden"}>
              <SamsaraWorkerSyncCard embedded open={open && activeView === "worker-sync"} />
            </Activity>
          </InnerContent>
        </SidebarProvider>
      </DialogContent>
    </Dialog>
  );
}

function InnerContent({ children }: { children: React.ReactNode }) {
  return <main className="flex min-w-0 flex-1 flex-col overflow-y-auto p-0">{children}</main>;
}
