import ChangePasswordForm from "@/app/auth/_components/change-password-form";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
import type { UserSchema } from "@/lib/schemas/user-schema";
import {
  faLock,
  faUser,
  IconDefinition,
} from "@fortawesome/pro-regular-svg-icons";
import { useState } from "react";
import { Icon } from "./ui/icons";
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
} from "./ui/sidebar";
import { UserProfileForm } from "./user-profile-form";

type UserSettingsTab = "profile" | "change-password";

interface UserSettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  user: UserSchema;
}

type UserSettingsTabItem = {
  name: UserSettingsTab;
  displayName: string;
  description: string;
  icon: IconDefinition;
  component: React.ReactNode;
};

const renderComponent = (
  user: UserSchema,
  onOpenChange: (open: boolean) => void,
) => {
  const nav: UserSettingsTabItem[] = [
    {
      name: "profile",
      displayName: "Profile",
      description:
        "Update your profile information to keep your account secure and personalized.",
      icon: faUser,
      component: <UserProfileForm user={user} />,
    },
    {
      name: "change-password",
      displayName: "Change Password",
      description: "Change your password to keep your account secure.",
      icon: faLock,
      component: <ChangePasswordForm onOpenChange={onOpenChange} />,
    },
  ];
  return nav;
};

export function UserSettingsDialog({
  open,
  onOpenChange,
  user,
}: UserSettingsDialogProps) {
  const [activeTab, setActiveTab] = useState<UserSettingsTab>("profile");
  const data = renderComponent(user, onOpenChange);
  const activeTabItem = data.find((nav) => nav.name === activeTab);

  const activeComponent = activeTabItem?.component as React.ReactNode;
  const activeTabName = activeTabItem?.displayName;
  const activeTabDescription = activeTabItem?.description;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="overflow-hidden p-0 md:max-h-[500px] md:max-w-[700px] lg:max-w-[800px]">
        <DialogTitle className="sr-only">Settings</DialogTitle>
        <DialogDescription className="sr-only">
          Customize your settings here.
        </DialogDescription>
        <SidebarProvider className="min-h-full items-start">
          <Sidebar collapsible="none" className="hidden md:flex">
            <SidebarContent>
              <SidebarGroup>
                <SidebarGroupContent>
                  <SidebarMenu>
                    {data.map((item) => (
                      <SidebarMenuItem key={item.name}>
                        <SidebarMenuButton
                          asChild
                          isActive={activeTab === item.name}
                          onClick={() =>
                            setActiveTab(item.name as UserSettingsTab)
                          }
                        >
                          <span>
                            <Icon icon={item.icon} />
                            <span>{item.displayName}</span>
                          </span>
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    ))}
                  </SidebarMenu>
                </SidebarGroupContent>
              </SidebarGroup>
            </SidebarContent>
          </Sidebar>
          <MainContentOuter>
            <header className="flex h-16 shrink-0 items-center gap-2 border-b border-border transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-12">
              <div className="flex flex-col items-start gap-0.5 px-4">
                <h1 className="text-lg font-medium">{activeTabName}</h1>
                <p className="text-sm text-muted-foreground">
                  {activeTabDescription}
                </p>
              </div>
            </header>
            <MainContent>{activeComponent}</MainContent>
          </MainContentOuter>
        </SidebarProvider>
      </DialogContent>
    </Dialog>
  );
}

function MainContent({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex flex-1 flex-col gap-4 pt-0">
      <div className="aspect-video max-w-3xl rounded-xl">{children}</div>
    </div>
  );
}

function MainContentOuter({ children }: { children: React.ReactNode }) {
  return (
    <main className="flex h-[480px] flex-1 flex-col overflow-hidden">
      {children}
    </main>
  );
}
