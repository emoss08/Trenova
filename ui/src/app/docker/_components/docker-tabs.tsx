/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
 */

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ContainerList } from "./container-list";
import { ImageList } from "./image-list";
import { NetworkList } from "./network-list";
import { VolumeList } from "./volume-list";

export function DockerTabs() {
  return (
    <Tabs defaultValue="containers" className="items-center">
      <TabsList className="h-auto rounded-none border-b bg-transparent p-0 w-full justify-start">
        <TabsTrigger
          value="containers"
          className="group data-[state=active]:after:bg-primary data-[state=active]:text-primary relative rounded-none py-2 after:absolute after:inset-x-0 after:bottom-0 after:h-0.5 data-[state=active]:bg-transparent data-[state=active]:shadow-none"
        >
          Containers
        </TabsTrigger>
        <TabsTrigger
          value="images"
          className="group data-[state=active]:after:bg-primary data-[state=active]:text-primary relative rounded-none py-2 after:absolute after:inset-x-0 after:bottom-0 after:h-0.5 data-[state=active]:bg-transparent data-[state=active]:shadow-none"
        >
          Images
        </TabsTrigger>
        <TabsTrigger
          value="volumes"
          className="group data-[state=active]:after:bg-primary data-[state=active]:text-primary relative rounded-none py-2 after:absolute after:inset-x-0 after:bottom-0 after:h-0.5 data-[state=active]:bg-transparent data-[state=active]:shadow-none"
        >
          Volumes
        </TabsTrigger>
        <TabsTrigger
          value="networks"
          className="group data-[state=active]:after:bg-primary data-[state=active]:text-primary relative rounded-none py-2 after:absolute after:inset-x-0 after:bottom-0 after:h-0.5 data-[state=active]:bg-transparent data-[state=active]:shadow-none"
        >
          Networks
        </TabsTrigger>
      </TabsList>

      <TabsContent value="containers" className="w-full">
        <ContainerList />
      </TabsContent>

      <TabsContent value="images" className="w-full">
        <ImageList />
      </TabsContent>

      <TabsContent value="volumes" className="w-full">
        <VolumeList />
      </TabsContent>

      <TabsContent value="networks" className="w-full">
        <NetworkList />
      </TabsContent>
    </Tabs>
  );
}
