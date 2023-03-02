import React from "react";
import Badge from "./Badges";

interface SystemHealthProps {
  service: string;
  status: string;
}

const statusToClass: Record<string, string> = {
  working: "bg-green-100",
  offline: "bg-red-100",
  // Add more status values as needed
};

function SystemHealth(props: SystemHealthProps) {
  const statusClass =
    statusToClass[props.status?.toLowerCase()] || "bg-gray-100";
  const statusTextColor =
    statusClass === "bg-red-100" ? "text-red-800" : "text-green-800";
  const statusDotColor =
    statusClass === "bg-red-100" ? "text-red-400" : "text-green-400";

  return (
    <div className="flex items-center space-x-3">
      <h3 className="truncate text-sm font-medium text-gray-900">
        {props.service}
      </h3>
      <Badge
        text={props.status}
        bgColor={statusClass}
        dotColor={statusDotColor}
        textColor={statusTextColor}
      />
    </div>
  );
}

export default SystemHealth;
