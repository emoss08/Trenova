import { useState } from "react";
import { useNotificationPreferences, useDeleteNotificationPreference } from "@/hooks/use-notifications";
import { NotificationPreference } from "@/types/notification";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import { Bell, BellOff, Clock, Edit, Plus, Trash2, Filter } from "lucide-react";
import { NotificationPreferencesForm } from "./notification-preferences-form";
import { cn } from "@/lib/utils";

export function NotificationPreferencesList() {
  const { data, isLoading } = useNotificationPreferences();
  const deleteMutation = useDeleteNotificationPreference();
  const [editingPreference, setEditingPreference] = useState<NotificationPreference | null>(null);
  const [deletingPreference, setDeletingPreference] = useState<NotificationPreference | null>(null);
  const [showCreateDialog, setShowCreateDialog] = useState(false);

  const handleDelete = () => {
    if (deletingPreference) {
      deleteMutation.mutate(deletingPreference.id, {
        onSuccess: () => {
          setDeletingPreference(null);
        },
      });
    }
  };

  if (isLoading) {
    return <NotificationPreferencesListSkeleton />;
  }

  const preferences = data?.data || [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Notification Preferences</h2>
          <p className="text-muted-foreground">
            Manage how you receive notifications for different types of updates
          </p>
        </div>
        <Button onClick={() => setShowCreateDialog(true)}>
          <Plus className="h-4 w-4 mr-2" />
          Add Preference
        </Button>
      </div>

      {preferences.length === 0 ? (
        <Card className="border-dashed">
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Bell className="h-12 w-12 text-muted-foreground mb-4" />
            <h3 className="text-lg font-medium">No notification preferences</h3>
            <p className="text-sm text-muted-foreground text-center max-w-sm mt-2">
              Create notification preferences to receive updates when records you own are modified
            </p>
            <Button className="mt-4" onClick={() => setShowCreateDialog(true)}>
              <Plus className="h-4 w-4 mr-2" />
              Create your first preference
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4">
          {preferences.map((preference) => (
            <NotificationPreferenceCard
              key={preference.id}
              preference={preference}
              onEdit={() => setEditingPreference(preference)}
              onDelete={() => setDeletingPreference(preference)}
            />
          ))}
        </div>
      )}

      {/* Create Dialog */}
      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Create Notification Preference</DialogTitle>
          </DialogHeader>
          <NotificationPreferencesForm
            onSuccess={() => setShowCreateDialog(false)}
            onCancel={() => setShowCreateDialog(false)}
          />
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog open={!!editingPreference} onOpenChange={(open) => !open && setEditingPreference(null)}>
        <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Edit Notification Preference</DialogTitle>
          </DialogHeader>
          {editingPreference && (
            <NotificationPreferencesForm
              preference={editingPreference}
              onSuccess={() => setEditingPreference(null)}
              onCancel={() => setEditingPreference(null)}
            />
          )}
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <AlertDialog open={!!deletingPreference} onOpenChange={(open) => !open && setDeletingPreference(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Notification Preference</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete this notification preference? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete} className="bg-destructive text-destructive-foreground">
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

interface NotificationPreferenceCardProps {
  preference: NotificationPreference;
  onEdit: () => void;
  onDelete: () => void;
}

function NotificationPreferenceCard({ preference, onEdit, onDelete }: NotificationPreferenceCardProps) {
  const updateTypeCount = preference.notifyOnAllUpdates 
    ? "All updates" 
    : `${preference.updateTypes.length} update type${preference.updateTypes.length !== 1 ? 's' : ''}`;

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between">
          <div className="space-y-1">
            <CardTitle className="flex items-center gap-2">
              {preference.resource.charAt(0).toUpperCase() + preference.resource.slice(1)}
              {!preference.isActive && (
                <Badge variant="secondary">
                  <BellOff className="h-3 w-3 mr-1" />
                  Inactive
                </Badge>
              )}
            </CardTitle>
            <CardDescription>
              {preference.notifyOnlyOwnedRecords ? "Only my records" : "All records"} · {updateTypeCount}
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            <Switch
              checked={preference.isActive}
              onCheckedChange={() => {
                // TODO: Implement quick toggle
              }}
            />
            <Button variant="ghost" size="icon" onClick={onEdit}>
              <Edit className="h-4 w-4" />
            </Button>
            <Button variant="ghost" size="icon" onClick={onDelete}>
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <div className="flex flex-wrap gap-2">
          {preference.quietHoursEnabled && (
            <Badge variant="outline">
              <Clock className="h-3 w-3 mr-1" />
              Quiet hours: {preference.quietHoursStart} - {preference.quietHoursEnd}
            </Badge>
          )}
          {preference.batchNotifications && (
            <Badge variant="outline">
              <Filter className="h-3 w-3 mr-1" />
              Batched every {preference.batchIntervalMinutes} min
            </Badge>
          )}
          {preference.excludedUserIds.length > 0 && (
            <Badge variant="outline">
              {preference.excludedUserIds.length} excluded user{preference.excludedUserIds.length !== 1 ? 's' : ''}
            </Badge>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

function NotificationPreferencesListSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <Skeleton className="h-8 w-64" />
          <Skeleton className="h-4 w-96 mt-2" />
        </div>
        <Skeleton className="h-10 w-32" />
      </div>
      <div className="grid gap-4">
        {[1, 2, 3].map((i) => (
          <Card key={i}>
            <CardHeader>
              <div className="flex items-start justify-between">
                <div className="space-y-2">
                  <Skeleton className="h-6 w-32" />
                  <Skeleton className="h-4 w-48" />
                </div>
                <div className="flex items-center gap-2">
                  <Skeleton className="h-6 w-10" />
                  <Skeleton className="h-8 w-8" />
                  <Skeleton className="h-8 w-8" />
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div className="flex gap-2">
                <Skeleton className="h-6 w-24" />
                <Skeleton className="h-6 w-32" />
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}