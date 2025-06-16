import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Input } from "@/components/ui/input";
import { 
  CreateNotificationPreferenceInput,
  UpdateNotificationPreferenceInput,
  NotificationPreference,
  UPDATE_TYPE_LABELS,
  NOTIFICATION_RESOURCES
} from "@/types/notification";
import { useCreateNotificationPreference, useUpdateNotificationPreference } from "@/hooks/use-notifications";
import { Clock, Bell, Users, Filter } from "lucide-react";

const notificationPreferenceSchema = z.object({
  resource: z.string().min(1, "Resource is required"),
  notifyOnAllUpdates: z.boolean().default(false),
  updateTypes: z.array(z.string()).optional(),
  preferredChannels: z.array(z.string()).min(1, "At least one channel is required"),
  quietHoursEnabled: z.boolean().default(false),
  quietHoursStart: z.string().optional(),
  quietHoursEnd: z.string().optional(),
  timezone: z.string().default(Intl.DateTimeFormat().resolvedOptions().timeZone),
  batchNotifications: z.boolean().default(false),
  batchIntervalMinutes: z.number().min(1).max(1440).default(15),
  excludedUserIds: z.array(z.string()).default([]),
  notifyOnlyOwnedRecords: z.boolean().default(true),
});

type FormData = z.infer<typeof notificationPreferenceSchema>;

interface NotificationPreferencesFormProps {
  preference?: NotificationPreference;
  onSuccess?: () => void;
  onCancel?: () => void;
}

export function NotificationPreferencesForm({ 
  preference, 
  onSuccess, 
  onCancel 
}: NotificationPreferencesFormProps) {
  const createMutation = useCreateNotificationPreference();
  const updateMutation = useUpdateNotificationPreference();
  
  const form = useForm<FormData>({
    resolver: zodResolver(notificationPreferenceSchema),
    defaultValues: preference ? {
      resource: preference.resource,
      notifyOnAllUpdates: preference.notifyOnAllUpdates,
      updateTypes: preference.updateTypes,
      preferredChannels: preference.preferredChannels,
      quietHoursEnabled: preference.quietHoursEnabled,
      quietHoursStart: preference.quietHoursStart,
      quietHoursEnd: preference.quietHoursEnd,
      timezone: preference.timezone,
      batchNotifications: preference.batchNotifications,
      batchIntervalMinutes: preference.batchIntervalMinutes,
      excludedUserIds: preference.excludedUserIds,
      notifyOnlyOwnedRecords: preference.notifyOnlyOwnedRecords,
    } : {
      resource: "",
      notifyOnAllUpdates: false,
      updateTypes: [],
      preferredChannels: ["user"],
      quietHoursEnabled: false,
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      batchNotifications: false,
      batchIntervalMinutes: 15,
      excludedUserIds: [],
      notifyOnlyOwnedRecords: true,
    },
  });

  const handleSubmit = (data: FormData) => {
    if (preference) {
      updateMutation.mutate(
        { 
          id: preference.id, 
          data: data as UpdateNotificationPreferenceInput 
        },
        {
          onSuccess: () => {
            onSuccess?.();
          },
        }
      );
    } else {
      createMutation.mutate(
        data as CreateNotificationPreferenceInput,
        {
          onSuccess: () => {
            onSuccess?.();
          },
        }
      );
    }
  };

  const isLoading = createMutation.isPending || updateMutation.isPending;

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle>Basic Settings</CardTitle>
            <CardDescription>
              Configure which updates you want to be notified about
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <FormField
              control={form.control}
              name="resource"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Resource Type</FormLabel>
                  <Select
                    disabled={!!preference}
                    value={field.value}
                    onValueChange={field.onChange}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select resource type" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {NOTIFICATION_RESOURCES.map((resource) => (
                        <SelectItem key={resource} value={resource}>
                          {resource.charAt(0).toUpperCase() + resource.slice(1)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormDescription>
                    The type of records you want to receive notifications for
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="notifyOnlyOwnedRecords"
              render={({ field }) => (
                <FormItem className="flex flex-row items-center justify-between rounded-lg border p-3 shadow-sm">
                  <div className="space-y-0.5">
                    <FormLabel>Only My Records</FormLabel>
                    <FormDescription>
                      Only receive notifications for records you created
                    </FormDescription>
                  </div>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="notifyOnAllUpdates"
              render={({ field }) => (
                <FormItem className="flex flex-row items-center justify-between rounded-lg border p-3 shadow-sm">
                  <div className="space-y-0.5">
                    <FormLabel>All Updates</FormLabel>
                    <FormDescription>
                      Receive notifications for all types of updates
                    </FormDescription>
                  </div>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </FormItem>
              )}
            />

            {!form.watch("notifyOnAllUpdates") && (
              <FormField
                control={form.control}
                name="updateTypes"
                render={() => (
                  <FormItem>
                    <div className="mb-4">
                      <FormLabel>Update Types</FormLabel>
                      <FormDescription>
                        Select which types of updates you want to be notified about
                      </FormDescription>
                    </div>
                    <div className="space-y-2">
                      {Object.entries(UPDATE_TYPE_LABELS).map(([type, label]) => (
                        <FormField
                          key={type}
                          control={form.control}
                          name="updateTypes"
                          render={({ field }) => {
                            return (
                              <FormItem
                                key={type}
                                className="flex flex-row items-start space-x-3 space-y-0"
                              >
                                <FormControl>
                                  <Checkbox
                                    checked={field.value?.includes(type)}
                                    onCheckedChange={(checked) => {
                                      return checked
                                        ? field.onChange([...(field.value || []), type])
                                        : field.onChange(
                                            field.value?.filter((value) => value !== type)
                                          );
                                    }}
                                  />
                                </FormControl>
                                <FormLabel className="font-normal">
                                  {label}
                                </FormLabel>
                              </FormItem>
                            );
                          }}
                        />
                      ))}
                    </div>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Delivery Settings</CardTitle>
            <CardDescription>
              Configure how and when you receive notifications
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Tabs defaultValue="timing" className="w-full">
              <TabsList className="grid w-full grid-cols-3">
                <TabsTrigger value="timing">
                  <Clock className="w-4 h-4 mr-2" />
                  Timing
                </TabsTrigger>
                <TabsTrigger value="batching">
                  <Bell className="w-4 h-4 mr-2" />
                  Batching
                </TabsTrigger>
                <TabsTrigger value="filters">
                  <Filter className="w-4 h-4 mr-2" />
                  Filters
                </TabsTrigger>
              </TabsList>

              <TabsContent value="timing" className="space-y-4 mt-4">
                <FormField
                  control={form.control}
                  name="quietHoursEnabled"
                  render={({ field }) => (
                    <FormItem className="flex flex-row items-center justify-between rounded-lg border p-3 shadow-sm">
                      <div className="space-y-0.5">
                        <FormLabel>Quiet Hours</FormLabel>
                        <FormDescription>
                          Pause notifications during specific hours
                        </FormDescription>
                      </div>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />

                {form.watch("quietHoursEnabled") && (
                  <div className="grid grid-cols-2 gap-4">
                    <FormField
                      control={form.control}
                      name="quietHoursStart"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Start Time</FormLabel>
                          <FormControl>
                            <Input
                              type="time"
                              {...field}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="quietHoursEnd"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>End Time</FormLabel>
                          <FormControl>
                            <Input
                              type="time"
                              {...field}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>
                )}

                <FormField
                  control={form.control}
                  name="timezone"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Timezone</FormLabel>
                      <FormControl>
                        <Input {...field} readOnly />
                      </FormControl>
                      <FormDescription>
                        Automatically detected from your browser
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </TabsContent>

              <TabsContent value="batching" className="space-y-4 mt-4">
                <FormField
                  control={form.control}
                  name="batchNotifications"
                  render={({ field }) => (
                    <FormItem className="flex flex-row items-center justify-between rounded-lg border p-3 shadow-sm">
                      <div className="space-y-0.5">
                        <FormLabel>Batch Notifications</FormLabel>
                        <FormDescription>
                          Group multiple notifications into summaries
                        </FormDescription>
                      </div>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />

                {form.watch("batchNotifications") && (
                  <FormField
                    control={form.control}
                    name="batchIntervalMinutes"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Batch Interval (minutes)</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            min={1}
                            max={1440}
                            {...field}
                            onChange={(e) => field.onChange(parseInt(e.target.value))}
                          />
                        </FormControl>
                        <FormDescription>
                          How often to send batched notifications (1-1440 minutes)
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                )}
              </TabsContent>

              <TabsContent value="filters" className="space-y-4 mt-4">
                <div className="space-y-4">
                  <div>
                    <Label>Excluded Users</Label>
                    <p className="text-sm text-muted-foreground mt-1">
                      You won't receive notifications when these users update your records
                    </p>
                    {/* TODO: Add user selector component */}
                    <div className="mt-2 p-4 border rounded-md text-center text-muted-foreground">
                      User selection coming soon
                    </div>
                  </div>
                </div>
              </TabsContent>
            </Tabs>
          </CardContent>
        </Card>

        <div className="flex justify-end gap-2">
          {onCancel && (
            <Button
              type="button"
              variant="outline"
              onClick={onCancel}
              disabled={isLoading}
            >
              Cancel
            </Button>
          )}
          <Button type="submit" disabled={isLoading}>
            {isLoading ? "Saving..." : preference ? "Update" : "Create"} Preference
          </Button>
        </div>
      </form>
    </Form>
  );
}