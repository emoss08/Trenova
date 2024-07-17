/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */



import { Notifications } from "@/components/layout/notification_menu/notification";
import { Button } from "@/components/ui/button";
import { ComponentLoader } from "@/components/ui/component-loader";
import { InternalLink } from "@/components/ui/link";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useNotifications } from "@/hooks/useQueries";
import axios from "@/lib/axiosConfig";
import { useUserStore } from "@/stores/AuthStore";
import { useHeaderStore } from "@/stores/HeaderStore";
import { UserNotification } from "@/types/accounts";
import { useQueryClient } from "@tanstack/react-query";
import { BellIcon } from "lucide-react";
import React, { useState } from "react";
import { toast } from "sonner";
import { ArchiveMenuContent } from "./archive-menu";
import { DownloadMenuContent } from "./download-menu";

function NotificationButton({
  userHasNotifications,
  open,
}: {
  userHasNotifications: boolean;
  open: boolean;
}) {
  return (
    <TooltipProvider delayDuration={100}>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            size="icon"
            variant="outline"
            aria-expanded={open}
            className="border-muted-foreground/40 hover:border-muted-foreground/80 group relative size-8"
          >
            <BellIcon
              strokeWidth="1.5"
              className="text-muted-foreground group-hover:text-foreground size-5"
            />
            <span className="sr-only">Notifications</span>
            {userHasNotifications && (
              <span className="absolute -right-1 -top-1 flex size-2.5">
                <span className="absolute inline-flex size-full animate-ping rounded-full bg-green-400 opacity-100"></span>
                <span className="ring-background relative inline-flex size-2.5 rounded-full bg-green-600 ring-1"></span>
              </span>
            )}
          </Button>
        </TooltipTrigger>
        <TooltipContent side="bottom" sideOffset={5}>
          <span>Notifications</span>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

function NotificationContent({
  notificationsData,
  notificationsLoading,
  userHasNotifications,
  readAllNotifications,
}: {
  notificationsData: UserNotification | undefined;
  notificationsLoading: boolean;
  userHasNotifications: boolean;
  readAllNotifications: () => void;
}) {
  if (notificationsLoading) {
    return <ComponentLoader className="h-80" />;
  }

  return (
    <>
      <ScrollArea className="h-80 w-full">
        <Notifications
          notification={notificationsData as UserNotification}
          notificationLoading={notificationsLoading}
        />
      </ScrollArea>
      {!userHasNotifications && (
        <div className="select-none items-center justify-center border-t pt-2 text-center text-xs">
          Know when you have new notifications by enabling text notifications in
          your{" "}
          <InternalLink to="/account/settings/">Account Settings</InternalLink>
        </div>
      )}
      {userHasNotifications && (
        <div className="flex items-center justify-center border-t pt-2 text-center">
          <Button
            onClick={readAllNotifications}
            variant="link"
            className="w-full"
          >
            Mark all as read
          </Button>
        </div>
      )}
    </>
  );
}

export function NotificationMenu() {
  const queryClient = useQueryClient();
  const [notificationsMenuOpen, setNotificationMenuOpen] = useHeaderStore.use(
    "notificationMenuOpen",
  );
  const [userHasNotifications, setUserHasNotifications] =
    useState<boolean>(false);
  const { id: userId } = useUserStore.get("user");
  const { notificationsData, notificationsLoading } = useNotifications(userId);
  const [activeTab, setActiveTab] = useState<string>("inbox");

  const markedAndInvalidate = async () => {
    await axios.get("/user-notifications/?markAsRead=true");
    await queryClient.invalidateQueries({
      queryKey: ["userNotifications", userId],
    });
  };

  const readAllNotifications = async () => {
    const sendNotificationRequest = markedAndInvalidate();

    // Fire Toast
    toast.promise(sendNotificationRequest, {
      loading: "Marking all notifications as read",
      success: "All notifications marked as read",
      error: "Failed to mark all notifications as read",
    });

    setNotificationMenuOpen(false);
  };

  React.useEffect(() => {
    // Using optional chaining to safely access unreadList
    if (
      notificationsData?.unreadList &&
      notificationsData?.unreadList?.length > 0
    ) {
      setUserHasNotifications(true);
    } else {
      setUserHasNotifications(false);
    }
  }, [notificationsData]);

  return (
    <Popover
      open={notificationsMenuOpen}
      onOpenChange={setNotificationMenuOpen}
    >
      <PopoverTrigger asChild>
        <span>
          <NotificationButton
            userHasNotifications={userHasNotifications}
            open={notificationsMenuOpen}
          />
        </span>
      </PopoverTrigger>
      <PopoverContent
        className="bg-popover w-96 p-2"
        sideOffset={10}
        alignOffset={-40}
        align="end"
      >
        <Tabs
          defaultValue="inbox"
          value={activeTab}
          className="w-full flex-1"
          onValueChange={setActiveTab}
        >
          <TabsList className="mx-auto space-x-4">
            <TabsTrigger
              isNotification={userHasNotifications}
              notificationCount={notificationsData?.unreadCount}
              value="inbox"
            >
              Inbox
            </TabsTrigger>
            <TabsTrigger value="downloads">Downloads</TabsTrigger>
            <TabsTrigger value="archive">Archive</TabsTrigger>
          </TabsList>
          <TabsContent value="inbox">
            <NotificationContent
              notificationsData={notificationsData as UserNotification}
              notificationsLoading={notificationsLoading}
              userHasNotifications={userHasNotifications}
              readAllNotifications={readAllNotifications}
            />
          </TabsContent>
          <TabsContent value="archive">
            <ArchiveMenuContent />
          </TabsContent>
          <TabsContent value="downloads">
            <DownloadMenuContent />
          </TabsContent>
        </Tabs>
      </PopoverContent>
    </Popover>
  );
}
