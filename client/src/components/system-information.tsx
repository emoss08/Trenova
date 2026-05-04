import { formatCurrentUserTime } from "@/lib/date";
import { useAuthStore } from "@/stores/auth-store";
import { Dot } from "lucide-react";
import React from "react";

export function SystemInformation() {
  return (
    <div className="ml-auto flex items-center text-center gap-1 px-3 font-table text-xs text-muted-foreground">
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
    <div className="flex flex-row gap-1 items-center text-center justify-center">
      <div className="size-1.5 rounded-full bg-green-500 mb-0.5" />
      <span className="text-xs text-muted-foreground">Systems nominal</span>
    </div>
  );
}
