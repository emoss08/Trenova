/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
 */

import { DockerOverview } from "./_components/docker-overview";
import { DockerTabs } from "./_components/docker-tabs";

export function DockerManagement() {
  return (
    <div className="flex flex-col gap-4 p-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Docker Management</h1>
          <p className="text-muted-foreground">
            Manage containers, images, volumes, and networks
          </p>
        </div>
      </div>

      <DockerOverview />
      <DockerTabs />
    </div>
  );
}
