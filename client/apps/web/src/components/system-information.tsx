import { formatCurrentUserTime } from "@trenova/shared/lib/date";
import { Dot } from "lucide-react";
import React from "react";

export function SystemInformation() {
  return (
    <div className="font-table text-muted-foreground ml-auto flex items-center gap-1 px-3 text-center text-xs">
      <SystemStatus />
      <Dot className="text-muted-foreground size-2.5" />
      <UserCurrentTime />
    </div>
  );
}

function UserCurrentTime() {
  const [currentTime, setCurrentTime] = React.useState(new Date());

  React.useEffect(() => {
    const interval = setInterval(() => {
      setCurrentTime(new Date());
    }, 1000);

    return () => clearInterval(interval);
  }, []);

  return formatCurrentUserTime(currentTime);
}

function SystemStatus() {
  return (
    <div className="flex flex-row items-center justify-center gap-1 text-center">
      <div className="mb-0.5 size-1.5 rounded-full bg-green-500" />
      <span className="text-muted-foreground text-xs">Systems nominal</span>
    </div>
  );
}
