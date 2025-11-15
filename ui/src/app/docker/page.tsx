import { MetaTags } from "@/components/meta-tags";
import { DockerOverview } from "./_components/docker-overview";
import { DockerTabs } from "./_components/docker-tabs";

export function DockerManagement() {
  return (
    <div className="flex flex-col gap-4 p-4">
      <MetaTags title="Docker Management" description="Docker Management" />
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
