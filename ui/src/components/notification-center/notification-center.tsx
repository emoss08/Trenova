import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  useNotificationActions,
  useNotificationCleanup,
  useNotificationHistory,
} from "@/hooks/use-notifications";
import { useWebNotifications } from "@/hooks/use-web-notifications";
import { useWebSocketStore } from "@/stores/websocket-store";
import { faBellOn, faBellRing } from "@fortawesome/pro-duotone-svg-icons";
import {
  faBell,
  faCheckDouble,
  faGear,
} from "@fortawesome/pro-regular-svg-icons";
import { useCallback, useState } from "react";
import { Link } from "react-router-dom";
import { Icon } from "../ui/icons";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "../ui/tooltip";
import { NotificationItem } from "./navigation-item";

export function NotificationCenter() {
  const [isOpen, setIsOpen] = useState(false);
  const { notifications } = useWebSocketStore();
  const { data: history } = useNotificationHistory({ limit: 50 });
  const { markAsRead, markAllAsRead, dismiss, handleNotificationClick } =
    useNotificationActions();
  const { isEnabled, enableNotifications, disableNotifications } =
    useWebNotifications();

  useNotificationCleanup();

  const allNotifications = [...notifications, ...(history?.results || [])]
    .filter(
      (notif, index, self) =>
        index === self.findIndex((n) => n.id === notif.id),
    )
    .sort((a, b) => b.createdAt - a.createdAt);

  const unreadNotifications = allNotifications.filter(
    (n) => !n.readAt && !n.dismissedAt,
  );

  const handleNotificationAction = useCallback(
    (notificationId: string, data?: any) => {
      handleNotificationClick(notificationId, data);
      setIsOpen(false);
    },
    [handleNotificationClick],
  );

  const handlePermissionChange = useCallback(async () => {
    if (isEnabled) {
      disableNotifications();
    } else {
      await enableNotifications();
    }
  }, [isEnabled, enableNotifications, disableNotifications]);

  return (
    <TooltipProvider>
      <Popover open={isOpen} onOpenChange={setIsOpen}>
        <Tooltip>
          <PopoverTrigger asChild>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="size-7 items-center relative"
              >
                <Icon icon={faBell} className="text-muted-foreground" />
                {unreadNotifications.length > 0 && (
                  <span className="absolute -right-1 -top-0.5 flex size-2">
                    <span className="absolute inline-flex size-full animate-ping rounded-full bg-green-400 opacity-100"></span>
                    <span className="ring-background relative inline-flex size-2 rounded-full bg-green-600 ring-1"></span>
                  </span>
                )}
              </Button>
            </TooltipTrigger>
          </PopoverTrigger>
          <TooltipContent>
            <p>Notifications</p>
          </TooltipContent>
        </Tooltip>
        <PopoverContent className="w-96 p-0" align="end">
          <div className="flex items-center justify-between px-4 py-2">
            <h3 className="font-semibold">Notifications</h3>
            <div className="flex items-center gap-2">
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="secondary"
                    size="icon"
                    onClick={handlePermissionChange}
                  >
                    <Icon icon={isEnabled ? faBellOn : faBellRing} />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>
                  {isEnabled ? "Disable notifications" : "Enable notifications"}
                </TooltipContent>
              </Tooltip>
              {unreadNotifications.length > 0 && (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="secondary"
                      size="icon"
                      onClick={() => markAllAsRead.mutate()}
                      className="size-8 [&_svg]:size-4"
                    >
                      <Icon icon={faCheckDouble} />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Mark all as read</TooltipContent>
                </Tooltip>
              )}
              <Tooltip>
                <TooltipTrigger asChild>
                  <Link to="/settings/notifications">
                    <Button variant="secondary" size="icon" className="size-8">
                      <Icon icon={faGear} />
                    </Button>
                  </Link>
                </TooltipTrigger>
                <TooltipContent>Settings</TooltipContent>
              </Tooltip>
            </div>
          </div>
          <Tabs defaultValue="unread" className="w-full">
            <TabsList className="h-auto rounded-none border-b gap-6 bg-transparent p-0 w-full justify-start">
              <TabsTrigger
                value="unread"
                className="group data-[state=active]:after:bg-primary data-[state=active]:text-primary relative rounded-none px-4 py-2 after:absolute after:inset-x-0 after:bottom-0 after:h-0.5 data-[state=active]:bg-transparent data-[state=active]:shadow-none"
              >
                Unread
                {unreadNotifications.length > 0 && (
                  <div className="text-xs text-muted-foreground bg-muted border border-border rounded-md px-1.5 items-center justify-center py-0.5 size-full group-data-[state=active]:bg-primary group-data-[state=active]:text-background">
                    {unreadNotifications.length}
                  </div>
                )}
              </TabsTrigger>
              <TabsTrigger
                value="all"
                className="group data-[state=active]:after:bg-primary data-[state=active]:text-primary relative rounded-none px-4 py-2 after:absolute after:inset-x-0 after:bottom-0 after:h-0.5 data-[state=active]:bg-transparent data-[state=active]:shadow-none"
              >
                All
              </TabsTrigger>
            </TabsList>
            <TabsContent value="unread" className="m-0">
              <ScrollArea className="h-[400px]">
                {unreadNotifications.length === 0 ? (
                  <div className="flex flex-col items-center justify-center h-[200px] text-muted-foreground">
                    <Icon icon={faBell} className="size-12 mb-2 opacity-20" />
                    <p className="text-sm">No unread notifications</p>
                  </div>
                ) : (
                  <div className="divide-y">
                    {unreadNotifications.map((notification) => (
                      <NotificationItem
                        key={notification.id}
                        notification={notification}
                        onAction={handleNotificationAction}
                        onMarkAsRead={() => markAsRead.mutate(notification.id)}
                        onDismiss={() => dismiss.mutate(notification.id)}
                      />
                    ))}
                  </div>
                )}
              </ScrollArea>
            </TabsContent>
            <TabsContent value="all" className="m-0">
              <ScrollArea className="h-[400px]">
                {allNotifications.length === 0 ? (
                  <div className="flex flex-col items-center justify-center h-[200px] text-muted-foreground">
                    <Icon icon={faBell} className="size-12 mb-2 opacity-20" />
                    <p className="text-sm">No notifications yet</p>
                  </div>
                ) : (
                  <div className="divide-y">
                    {allNotifications.map((notification) => (
                      <NotificationItem
                        key={notification.id}
                        notification={notification}
                        onAction={handleNotificationAction}
                        onMarkAsRead={() => markAsRead.mutate(notification.id)}
                        onDismiss={() => dismiss.mutate(notification.id)}
                      />
                    ))}
                  </div>
                )}
              </ScrollArea>
            </TabsContent>
          </Tabs>
        </PopoverContent>
      </Popover>
    </TooltipProvider>
  );
}
