import "@fontsource-variable/newsreader/opsz-italic.css";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { user as userQueries, userOrganization } from "@/lib/queries/user";
import { greetingPeriod, hourIn } from "@/lib/user-settings";
import { useQuery } from "@tanstack/react-query";
import { BellOff01Icon, Camera01Icon, ClockIcon } from "@trenova/shared/components/icons";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime, resolveUserTimezone } from "@trenova/shared/lib/date";
import { formatTimezoneLabel } from "@trenova/shared/lib/timezones";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { AvailabilityStatus } from "@trenova/shared/types/user";
import type { ReactNode } from "react";
import type { ProfilePictureControls } from "./use-profile-picture";

const STATUS_DOT: Record<AvailabilityStatus, string> = {
  available: "bg-success ring-success-subtle",
  focus: "bg-brand ring-brand-subtle",
  away: "bg-warning ring-warning-subtle",
};

export function effectiveAvailability(
  status: AvailabilityStatus | undefined,
  focusUntil: number | null | undefined,
  awayBackOn: number | null | undefined,
  nowSeconds: number,
): AvailabilityStatus {
  if (status === "focus") {
    return focusUntil && focusUntil > nowSeconds ? "focus" : "available";
  }
  if (status === "away") {
    return !awayBackOn || awayBackOn > nowSeconds ? "away" : "available";
  }
  return "available";
}

function Greeting({ firstName, timezone }: { firstName: string; timezone: string }) {
  const rt = useRichT();
  const now = useNow("minute");
  const tags = {
    em: (children: ReactNode) => (
      <em className="font-newsreader font-normal tracking-normal italic">{children}</em>
    ),
  };
  const period = greetingPeriod(hourIn(new Date(now * 1000), timezone));

  switch (period) {
    case "late":
      return rt("Working late, <em>{0}</em>.", tags, firstName);
    case "morning":
      return rt("Good morning, <em>{0}</em>.", tags, firstName);
    case "afternoon":
      return rt("Good afternoon, <em>{0}</em>.", tags, firstName);
    case "evening":
      return rt("Good evening, <em>{0}</em>.", tags, firstName);
  }
}

export function SettingsHero({
  picture,
  onStatusClick,
}: {
  picture: ProfilePictureControls;
  onStatusClick: () => void;
}) {
  const t = useT();
  const user = useAuthStore((state) => state.user);
  const now = useNow("minute");
  const { data: organizations } = useQuery(userOrganization.all());
  const { data: preferences } = useQuery({
    ...userQueries.myNotificationPreferences(),
    staleTime: 30_000,
  });

  const timezone = resolveUserTimezone(user?.timezone);
  const organization = organizations?.find((org) => org.isCurrent)?.name;
  const firstName = user?.name?.trim().split(/\s+/)[0] || user?.username || "";
  const status = effectiveAvailability(
    user?.availabilityStatus,
    user?.focusUntil,
    user?.awayBackOn,
    now,
  );
  const paused = Boolean(preferences?.pausedUntil && preferences.pausedUntil > now);
  const statusLabel =
    status === "focus" ? t("Focusing") : status === "away" ? t("Away") : t("Available");

  return (
    <div className="flex flex-wrap items-center gap-5 pt-1 pb-9 min-[620px]:flex-nowrap">
      <div className="group/hero-avatar relative size-16 shrink-0">
        <ResolvedUserAvatar
          userId={user?.id}
          name={user?.name}
          profilePicUrl={user?.profilePicUrl}
          thumbnailUrl={user?.thumbnailUrl}
          variant="full"
          className="size-16"
        />
        <button
          type="button"
          onClick={picture.openPicker}
          aria-label={t("Change photo")}
          className="ui-focus-ring bg-scrim text-foreground-on-solid absolute inset-0 grid place-items-center rounded-full opacity-0 transition-opacity duration-150 group-hover/hero-avatar:opacity-100 focus-visible:opacity-100"
        >
          <Camera01Icon className="size-4.5" />
        </button>
      </div>
      <div className="min-w-0 flex-1">
        <h1 className="text-4xl leading-tight font-medium tracking-tight text-balance">
          <Greeting firstName={firstName} timezone={timezone} />
        </h1>
        <p className="text-foreground-subtle mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
          <span>{[user?.jobTitle || t("No title"), organization].filter(Boolean).join(" · ")}</span>
          <span className="flex items-center gap-1.5">
            <ClockIcon className="size-3" />
            {t(
              "{0} in {1}",
              formatUnixTime(now, { timezone, timeFormat: user?.timeFormat }),
              formatTimezoneLabel(timezone),
            )}
          </span>
          {paused ? (
            <span className="flex items-center gap-1.5">
              <BellOff01Icon className="size-3" />
              {t("Notifications paused")}
            </span>
          ) : null}
        </p>
      </div>
      <button
        type="button"
        onClick={onStatusClick}
        className="ui-focus-ring bg-card hover:bg-surface-hover inline-flex h-7 items-center gap-2 rounded-md border px-2.5 text-sm font-medium whitespace-nowrap transition-colors"
      >
        <span aria-hidden className={cn("size-2 rounded-full ring-3", STATUS_DOT[status])} />
        {statusLabel}
      </button>
    </div>
  );
}
