import { useState } from "react";
import { useNotificationHistory, useNotificationActions } from "@/hooks/use-notifications";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Bell, Check, CheckCheck, Clock, Filter, X } from "lucide-react";
import { PRIORITY_CONFIG, UPDATE_TYPE_LABELS } from "@/types/notification";
import { formatDistanceToNow } from "date-fns";
import { cn } from "@/lib/utils";

export default function NotificationHistoryPage() {
  const [resource, setResource] = useState<string | undefined>();
  const [priority, setPriority] = useState<string | undefined>();
  const [showUnreadOnly, setShowUnreadOnly] = useState(false);
  
  const { data, isLoading, isFetchingNextPage, fetchNextPage, hasNextPage } = useNotificationHistory({
    resource,
    priority,
    unreadOnly: showUnreadOnly,
    limit: 50,
  });

  const { markAsRead, markAllAsRead, dismiss } = useNotificationActions();

  const notifications = data?.data || [];
  const totalCount = data?.totalCount || 0;

  return (
    <div className="container mx-auto py-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Notification History</h1>
          <p className="text-muted-foreground mt-2">
            View and manage all your notifications
          </p>
        </div>
        <div className="flex items-center gap-2">
          {notifications.some(n => !n.readAt) && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => markAllAsRead.mutate()}
            >
              <CheckCheck className="h-4 w-4 mr-2" />
              Mark all as read
            </Button>
          )}
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Filters</CardTitle>
          <CardDescription>
            Filter notifications by type and priority
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-4">
            <div className="flex items-center gap-2">
              <Filter className="h-4 w-4 text-muted-foreground" />
              <Select value={resource || "all"} onValueChange={(v) => setResource(v === "all" ? undefined : v)}>
                <SelectTrigger className="w-[180px]">
                  <SelectValue placeholder="All resources" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All resources</SelectItem>
                  <SelectItem value="shipment">Shipments</SelectItem>
                  <SelectItem value="worker">Workers</SelectItem>
                  <SelectItem value="customer">Customers</SelectItem>
                  <SelectItem value="tractor">Tractors</SelectItem>
                  <SelectItem value="trailer">Trailers</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <Select value={priority || "all"} onValueChange={(v) => setPriority(v === "all" ? undefined : v)}>
              <SelectTrigger className="w-[180px]">
                <SelectValue placeholder="All priorities" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All priorities</SelectItem>
                <SelectItem value="critical">Critical</SelectItem>
                <SelectItem value="high">High</SelectItem>
                <SelectItem value="medium">Medium</SelectItem>
                <SelectItem value="low">Low</SelectItem>
              </SelectContent>
            </Select>

            <Button
              variant={showUnreadOnly ? "default" : "outline"}
              size="sm"
              onClick={() => setShowUnreadOnly(!showUnreadOnly)}
            >
              <Bell className="h-4 w-4 mr-2" />
              Unread only
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>
            Notifications
            {totalCount > 0 && (
              <span className="text-muted-foreground font-normal text-sm ml-2">
                ({totalCount} total)
              </span>
            )}
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {isLoading ? (
            <NotificationHistorySkeleton />
          ) : notifications.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
              <Bell className="h-12 w-12 mb-4 opacity-20" />
              <p className="text-lg font-medium">No notifications found</p>
              <p className="text-sm mt-1">Try adjusting your filters</p>
            </div>
          ) : (
            <ScrollArea className="h-[600px]">
              <div className="divide-y">
                {notifications.map((notification) => (
                  <NotificationHistoryItem
                    key={notification.id}
                    notification={notification}
                    onMarkAsRead={() => markAsRead.mutate(notification.id)}
                    onDismiss={() => dismiss.mutate(notification.id)}
                  />
                ))}
              </div>
              {hasNextPage && (
                <div className="p-4 text-center">
                  <Button
                    variant="outline"
                    onClick={() => fetchNextPage()}
                    disabled={isFetchingNextPage}
                  >
                    {isFetchingNextPage ? "Loading..." : "Load more"}
                  </Button>
                </div>
              )}
            </ScrollArea>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

interface NotificationHistoryItemProps {
  notification: any;
  onMarkAsRead: () => void;
  onDismiss: () => void;
}

function NotificationHistoryItem({ 
  notification, 
  onMarkAsRead, 
  onDismiss 
}: NotificationHistoryItemProps) {
  const priorityConfig = PRIORITY_CONFIG[notification.priority];

  return (
    <div
      className={cn(
        "px-6 py-4 hover:bg-muted/50 transition-colors relative group",
        !notification.readAt && "bg-muted/20"
      )}
    >
      <div className="flex gap-4">
        <div className={cn(
          "h-2 w-2 rounded-full mt-2 flex-shrink-0",
          priorityConfig.bgColor,
          !notification.readAt && "animate-pulse"
        )} />
        
        <div className="flex-1 min-w-0">
          <div className="flex items-start justify-between gap-2">
            <div className="flex-1">
              <h4 className={cn(
                "font-medium text-sm",
                notification.readAt && "text-muted-foreground"
              )}>
                {notification.title}
              </h4>
              <p className={cn(
                "text-sm mt-1",
                notification.readAt ? "text-muted-foreground" : "text-foreground"
              )}>
                {notification.message}
              </p>
            </div>
            <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
              {!notification.readAt && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8"
                  onClick={onMarkAsRead}
                >
                  <Check className="h-4 w-4" />
                </Button>
              )}
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8"
                onClick={onDismiss}
              >
                <X className="h-4 w-4" />
              </Button>
            </div>
          </div>
          
          <div className="flex items-center gap-2 mt-2 flex-wrap">
            <Badge variant="outline" className="text-xs">
              {notification.eventType}
            </Badge>
            {notification.resource && (
              <Badge variant="outline" className="text-xs">
                {notification.resource}
              </Badge>
            )}
            {notification.updateType && UPDATE_TYPE_LABELS[notification.updateType] && (
              <Badge variant="outline" className="text-xs">
                {UPDATE_TYPE_LABELS[notification.updateType]}
              </Badge>
            )}
            <span className="text-xs text-muted-foreground flex items-center gap-1">
              <Clock className="h-3 w-3" />
              {formatDistanceToNow(notification.createdAt * 1000, { addSuffix: true })}
            </span>
            {notification.deliveryStatus === "failed" && (
              <Badge variant="destructive" className="text-xs">
                Failed
              </Badge>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function NotificationHistorySkeleton() {
  return (
    <div className="divide-y">
      {[1, 2, 3, 4, 5].map((i) => (
        <div key={i} className="px-6 py-4">
          <div className="flex gap-4">
            <Skeleton className="h-2 w-2 rounded-full mt-2" />
            <div className="flex-1 space-y-2">
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-4 w-full" />
              <div className="flex gap-2 mt-2">
                <Skeleton className="h-5 w-20" />
                <Skeleton className="h-5 w-24" />
                <Skeleton className="h-5 w-32" />
              </div>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}