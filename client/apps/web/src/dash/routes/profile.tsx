import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Switch } from "@trenova/shared/components/ui/switch";
import { useTheme } from "@trenova/shared/components/theme-provider";
import { disablePush, enablePush, getPushSubscription, pushSupported } from "@trenova/shared/lib/push";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { BellRingIcon, LogOutIcon, MoonIcon, SunIcon, MonitorIcon } from "lucide-react";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import { ComplianceCard } from "../_components/compliance-card";
import { useDashProfile } from "../_components/dash-layout";
import { ProfileDocuments } from "../_components/profile-documents";
import { PtoSection } from "../_components/pto-section";
import { useDashFeatures } from "../_components/use-dash-features";
import { cn } from "@trenova/shared/lib/utils";

const themeOptions = [
  { value: "light", label: "Light", icon: SunIcon },
  { value: "dark", label: "Dark", icon: MoonIcon },
  { value: "system", label: "Auto", icon: MonitorIcon },
] as const;

export function DashProfilePage() {
  const navigate = useNavigate();
  const logout = useAuthStore((state) => state.logout);
  const { theme, setTheme } = useTheme();
  const profile = useDashProfile();
  const features = useDashFeatures();

  const handleLogout = async () => {
    await logout();
    void navigate("/dash/login", { replace: true });
  };

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-xl font-semibold tracking-tight">Profile</h1>

      {profile.isPending ? (
        <Skeleton className="h-40 w-full rounded-2xl" />
      ) : profile.data ? (
        <div className="rounded-2xl border border-border bg-card p-4">
          <p className="text-lg font-semibold">
            {profile.data.firstName} {profile.data.lastName}
          </p>
          <p className="text-sm text-muted-foreground">{profile.data.organizationName}</p>
          <dl className="mt-3 flex flex-col gap-1.5 border-t border-border pt-3 text-sm">
            <ProfileRow label="Email" value={profile.data.email} />
            <ProfileRow label="Phone" value={profile.data.phoneNumber} />
            <ProfileRow label="Driver type" value={profile.data.driverType} />
            <ProfileRow label="Classification" value={profile.data.workerType} />
            <ProfileRow label="Fleet" value={profile.data.fleetCodeName} />
          </dl>
        </div>
      ) : (
        <div className="rounded-2xl border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
          We couldn&apos;t load your profile.
        </div>
      )}

      <ComplianceCard />

      <ProfileDocuments />

      {features.allowPtoRequests ? <PtoSection /> : null}

      <PushNotificationsCard />

      <div className="rounded-2xl border border-border bg-card p-4">
        <p className="text-sm font-semibold">Appearance</p>
        <div className="mt-3 grid grid-cols-3 gap-2">
          {themeOptions.map((option) => (
            <button
              key={option.value}
              type="button"
              onClick={() => setTheme(option.value)}
              className={cn(
                "flex flex-col items-center gap-1 rounded-lg border border-border py-3 text-xs font-medium text-muted-foreground",
                theme === option.value && "border-primary text-foreground",
              )}
            >
              <option.icon className="size-4" />
              {option.label}
            </button>
          ))}
        </div>
      </div>

      <Button variant="outline" className="h-11" onClick={handleLogout}>
        <LogOutIcon className="size-4" />
        Sign out
      </Button>

      <p className="text-center text-xs text-muted-foreground">
        Questions about your pay? Flag it on the statement or call your fleet manager.
      </p>
    </div>
  );
}

function ProfileRow({ label, value }: { label: string; value: string }) {
  if (!value) {
    return null;
  }
  return (
    <div className="flex items-center justify-between gap-4">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="truncate font-medium">{value}</dd>
    </div>
  );
}

function PushNotificationsCard() {
  const queryClient = useQueryClient();
  const supported = pushSupported();

  const subscription = useQuery({
    queryKey: ["dash-push-subscription"],
    queryFn: async () => {
      const existing = await getPushSubscription();
      return existing != null;
    },
    enabled: supported,
  });

  const toggle = useMutation({
    mutationFn: async (enable: boolean) => {
      if (enable) {
        await enablePush();
      } else {
        await disablePush();
      }
      return enable;
    },
    onSuccess: async (enabled) => {
      toast.success(
        enabled
          ? "Push notifications are on — you'll hear about loads and pay even with Dash closed."
          : "Push notifications turned off.",
      );
      await queryClient.invalidateQueries({ queryKey: ["dash-push-subscription"] });
    },
    onError: (error: Error) =>
      toast.error(error.message || "We couldn't update push notifications."),
  });

  return (
    <div className="rounded-2xl border border-border bg-card p-4">
      <div className="flex items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2">
          <BellRingIcon className="size-4 shrink-0 text-muted-foreground" />
          <div className="min-w-0">
            <p className="text-sm font-semibold">Push notifications</p>
            <p className="text-xs text-muted-foreground">
              {supported
                ? "Load assignments, settlements, and pay updates — even when Dash is closed."
                : "Not supported in this browser. On iPhone, add Dash to your Home Screen first."}
            </p>
          </div>
        </div>
        <Switch
          checked={subscription.data === true}
          disabled={!supported || subscription.isPending || toggle.isPending}
          onCheckedChange={(checked) => toggle.mutate(checked)}
        />
      </div>
    </div>
  );
}
