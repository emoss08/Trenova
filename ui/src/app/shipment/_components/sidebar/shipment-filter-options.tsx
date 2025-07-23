/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { Button } from "@/components/ui/button";
import { useState } from "react";

type TabType = "shipments" | "vehicles" | "assignments" | "assets";

const TABS: { id: TabType; label: string }[] = [
  { id: "shipments", label: "Shipments" },
  { id: "vehicles", label: "Vehicles" },
  { id: "assignments", label: "Assignments" },
  { id: "assets", label: "Assets" },
];

export function FilterOptions() {
  const [activeTab, setActiveTab] = useState<TabType>("shipments");

  return (
    <div className="flex flex-row gap-2 justify-start">
      {TABS.map(({ id, label }) => (
        <Button
          key={id}
          variant={activeTab === id ? "default" : "secondary"}
          size="sm"
          onClick={() => setActiveTab(id)}
        >
          {label}
        </Button>
      ))}
    </div>
  );
}
