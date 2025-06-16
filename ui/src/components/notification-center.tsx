import { Bell, Check, CheckCheck, ChevronRight, Settings2, X } from "lucide-react";
import { useCallback, useState } from "react";
import { Link } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { 
  Popover, 
  PopoverContent, 
  PopoverTrigger 
} from "@/components/ui/popover";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { 
  useUnreadCount, 
  useNotificationHistory, 
  useNotificationActions,
  useNotificationCleanup 
} from "@/hooks/use-notifications";
import { useWebSocketStore } from "@/stores/websocket-store";
import { PRIORITY_CONFIG } from "@/types/notification";
import { formatDistanceToNow } from "date-fns";
import { cn } from "@/lib/utils";

export function NotificationCenter() {
  const [isOpen, setIsOpen] = useState(false);
  const { count: unreadCount } = useUnreadCount();
  const { notifications } = useWebSocketStore();
  const { data: history, isLoading } = useNotificationHistory({ limit: 50 });
  const { markAsRead, markAllAsRead, dismiss, handleNotificationClick } = useNotificationActions();

  // Clean up expired notifications
  useNotificationCleanup();

  const allNotifications = [
    ...notifications,
    ...(history?.data || [])
  ].filter((notif, index, self) => 
    index === self.findIndex((n) => n.id === notif.id)
  ).sort((a, b) => b.createdAt - a.createdAt);

  const unreadNotifications = allNotifications.filter(n => !n.readAt && !n.dismissedAt);
  const readNotifications = allNotifications.filter(n => n.readAt || n.dismissedAt);

  const handleNotificationAction = useCallback((notificationId: string, data?: any) => {
    handleNotificationClick(notificationId, data);
    setIsOpen(false);
  }, [handleNotificationClick]);

  return (
    <Popover open={isOpen} onOpenChange={setIsOpen}>
      <PopoverTrigger asChild>
        <Button variant="ghost" size="icon" className="relative">
          <Bell className="h-5 w-5" />
          {unreadCount > 0 && (
            <Badge 
              variant="destructive" 
              className="absolute -top-1 -right-1 h-5 w-5 rounded-full p-0 flex items-center justify-center"
            >
              {unreadCount > 99 ? "99+" : unreadCount}
            </Badge>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-96 p-0" align="end">
        <div className="flex items-center justify-between px-4 py-3 border-b">
          <h3 className="font-semibold">Notifications</h3>
          <div className="flex items-center gap-2">
            {unreadCount > 0 && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => markAllAsRead.mutate()}
                className="h-8 px-2 text-xs"
              >
                <CheckCheck className="h-3 w-3 mr-1" />
                Mark all as read
              </Button>
            )}
            <Link to="/settings/notifications">
              <Button variant="ghost" size="icon" className="h-8 w-8">
                <Settings2 className="h-4 w-4" />
              </Button>
            </Link>
          </div>
        </div>

        <Tabs defaultValue="unread" className="w-full">
          <TabsList className="grid w-full grid-cols-2 px-4 py-1">
            <TabsTrigger value="unread">
              Unread {unreadCount > 0 && `(${unreadCount})`}
            </TabsTrigger>
            <TabsTrigger value="all">All</TabsTrigger>
          </TabsList>

          <TabsContent value="unread" className="m-0">
            <ScrollArea className="h-[400px]">
              {unreadNotifications.length === 0 ? (
                <div className="flex flex-col items-center justify-center h-[200px] text-muted-foreground">
                  <Bell className="h-12 w-12 mb-2 opacity-20" />
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
                  <Bell className="h-12 w-12 mb-2 opacity-20" />
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

        <Separator />
        <div className="px-4 py-3">
          <Link to="/notifications/history">
            <Button variant="ghost" className="w-full justify-between">
              View all notifications
              <ChevronRight className="h-4 w-4" />
            </Button>
          </Link>
        </div>
      </PopoverContent>
    </Popover>
  );
}

interface NotificationItemProps {
  notification: any;
  onAction: (id: string, data?: any) => void;
  onMarkAsRead: () => void;
  onDismiss: () => void;
}

function NotificationItem({ 
  notification, 
  onAction, 
  onMarkAsRead, 
  onDismiss 
}: NotificationItemProps) {
  const priorityConfig = PRIORITY_CONFIG[notification.priority];

  return (
    <div
      className={cn(
        "px-4 py-3 hover:bg-muted/50 cursor-pointer transition-colors relative group",
        !notification.readAt && "bg-muted/20"
      )}
      onClick={() => onAction(notification.id, notification.data)}
    >
      <div className="flex gap-3">
        <div className={cn(
          "h-2 w-2 rounded-full mt-2 flex-shrink-0",
          priorityConfig.bgColor.replace("bg-", "bg-"),
          !notification.readAt && "animate-pulse"
        )} />
        
        <div className="flex-1 min-w-0">
          <div className="flex items-start justify-between gap-2">
            <h4 className={cn(
              "font-medium text-sm",
              notification.readAt && "text-muted-foreground"
            )}>
              {notification.title}
            </h4>
            <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
              {!notification.readAt && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6"
                  onClick={(e) => {
                    e.stopPropagation();
                    onMarkAsRead();
                  }}
                >
                  <Check className="h-3 w-3" />
                </Button>
              )}
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={(e) => {
                  e.stopPropagation();
                  onDismiss();
                }}
              >
                <X className="h-3 w-3" />
              </Button>
            </div>
          </div>
          
          <p className={cn(
            "text-sm mt-1",
            notification.readAt ? "text-muted-foreground" : "text-foreground"
          )}>
            {notification.message}
          </p>
          
          <div className="flex items-center gap-2 mt-2">
            <Badge variant="outline" className="text-xs">
              {notification.eventType}
            </Badge>
            <span className="text-xs text-muted-foreground">
              {formatDistanceToNow(notification.createdAt * 1000, { addSuffix: true })}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}