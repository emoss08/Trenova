import { formatCurrentUserTime } from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Dot } from "lucide-react";
import React from "react";

export function SystemInformation() {
  return (
    <div className="ml-auto flex items-center gap-1 px-3 text-center font-table text-xs text-muted-foreground">
      <SystemStatus />
      <Dot className="size-2.5 text-muted-foreground" />
      <UserCurrentTime />
    </div>
  );
}

function UserCurrentTime() {
  const user = useAuthStore((state) => state.user);
  const [currentTime, setCurrentTime] = React.useState(new Date());

  React.useEffect(() => {
    const interval = setInterval(() => {
      setCurrentTime(new Date());
    }, 1000);

    return () => clearInterval(interval);
  }, []);

  return formatCurrentUserTime(currentTime, user?.timeFormat, user?.timezone);
}

function SystemStatus() {
  return (
    <div className="flex flex-row items-center justify-center gap-1 text-center">
      <div className="mb-0.5 size-1.5 rounded-full bg-green-500" />
      <span className="text-xs text-muted-foreground">Systems nominal</span>
    </div>
  );
}
