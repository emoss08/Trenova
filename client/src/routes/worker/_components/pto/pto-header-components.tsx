import { timezoneChoices } from "@/lib/choices";
import { useAuthStore } from "@/stores/auth-store";
import type { User } from "@/types/user";

function HeaderOuter({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-[42px] items-center justify-between">{children}</div>
  );
}

function HeaderInner({ children }: { children: React.ReactNode }) {
  return <div className="flex h-full items-center gap-2">{children}</div>;
}

function getUserTimezoneLabel({ user }: { user: NonNullable<User> }) {
  return (
    timezoneChoices.find((choice) => choice.value === user.timezone)?.label ||
    "Auto-detect"
  );
}

export function HeaderContent({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  const user = useAuthStore((state) => state.user);
  const userTimezoneLabel = getUserTimezoneLabel({ user: user! });

  return (
    <HeaderOuter>
      <div className="flex flex-col leading-tight">
        <h3 className="font-table text-lg font-medium">{title}</h3>
        <p className="text-xs text-muted-foreground">
          Records shown in the timezone of <span>{userTimezoneLabel}</span>
        </p>
      </div>
      <HeaderInner>{children}</HeaderInner>
    </HeaderOuter>
  );
}
