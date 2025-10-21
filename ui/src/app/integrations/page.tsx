/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { RainbowButton } from "@/components/ui/rainbow-button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { IntegrationCategory } from "@/types/integration";
import {
  Map,
  Puzzle,
  RotateCcw,
  SatelliteDish,
  Search,
  Sparkles,
  Truck,
} from "lucide-react";
import { useMemo, useState } from "react";
import { IntegrationSkeleton } from "./_components/integration-skeleton";
import { getCategoryDisplayName } from "./_utils/integration";

export function IntegrationsPage() {
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState<"All" | IntegrationCategory>("All");

  const categories = useMemo(
    () =>
      [
        "All",
        IntegrationCategory.MappingRouting,
        IntegrationCategory.FreightLogistics,
        IntegrationCategory.Telematics,
      ] as const,
    [],
  );

  return (
    <>
      <MetaTags title="Apps & Integrations" />
      <section className="relative overflow-hidden rounded-2xl border border-input bg-card p-0">
        <div className="pointer-events-none absolute inset-0">
          <div className="absolute -left-16 -top-16 h-64 w-64 rounded-full blur-3xl opacity-40 bg-gradient-to-br from-pink-400/40 to-fuchsia-700/0" />
          <div className="absolute -right-24 -top-12 h-72 w-72 rounded-full blur-3xl opacity-40 bg-gradient-to-tr from-sky-300/35 via-blue-400/25 to-indigo-900/0" />
          <div className="absolute left-1/2 -bottom-24 h-80 w-80 -translate-x-1/2 rounded-full blur-3xl opacity-40 bg-gradient-to-b from-emerald-300/30 to-transparent" />
        </div>

        <div className="relative flex flex-col gap-3 p-6 md:p-8">
          <div className="inline-flex items-center gap-2">
            <Badge variant="indigo" withDot={false} className="h-6">
              <Sparkles className="size-4 text-indigo-500" />
              Marketplace
            </Badge>
          </div>
          <h1 className="text-2xl md:text-3xl font-semibold tracking-tight">
            Apps & Integrations
          </h1>
          <p className="text-sm md:text-base text-muted-foreground max-w-2xl">
            Extend Trenova with mapping, telematics, and logistics tools.
            Discover, enable, and manage integrations all in one place.
          </p>
          <div className="mt-2 flex flex-col gap-3 md:mt-4 md:flex-row md:items-center">
            <div className="flex-1">
              <Input
                placeholder="Search integrations by name, vendor, or description"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                icon={<Search className="text-muted-foreground size-4" />}
                className="h-9 text-sm"
              />
            </div>
            <div className="flex items-center gap-2">
              {search && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setSearch("")}
                  className="gap-2"
                >
                  <RotateCcw className="size-4" />
                  Reset
                </Button>
              )}
              <RainbowButton asChild size="lg">
                <a
                  target="_blank"
                  href="https://github.com/emoss08/Trenova/issues/new?assignees=&labels=enhancement&projects=&template=feature_request.md&title=Feature+Request%3A+%5BFeature+Name%5D"
                  rel="noreferrer"
                >
                  <Puzzle className="size-4" /> Suggest Integration
                </a>
              </RainbowButton>
            </div>
          </div>
          <div className="mt-1">
            <Tabs value={category} onValueChange={(v) => setCategory(v as any)}>
              <div className="flex items-center justify-between gap-2">
                <TabsList className="text-foreground mb-3 h-auto gap-2 w-full rounded-none border-b bg-transparent px-0 py-1 justify-start overflow-x-auto">
                  {categories.map((c) => (
                    <TabsTrigger
                      key={c}
                      value={c}
                      className="hover:bg-accent hover:text-foreground data-[state=active]:after:bg-primary data-[state=active]:hover:bg-accent relative after:absolute after:inset-x-0 after:bottom-0 after:-mb-1 after:h-0.5 data-[state=active]:bg-transparent data-[state=active]:shadow-none"
                    >
                      {c === "All" && (
                        <Puzzle
                          className="-ms-0.5 me-1.5 opacity-60"
                          size={16}
                          aria-hidden="true"
                        />
                      )}
                      {c === IntegrationCategory.MappingRouting && (
                        <Map
                          className="-ms-0.5 me-1.5 opacity-60"
                          size={16}
                          aria-hidden="true"
                        />
                      )}
                      {c === IntegrationCategory.FreightLogistics && (
                        <Truck
                          className="-ms-0.5 me-1.5 opacity-60"
                          size={16}
                          aria-hidden="true"
                        />
                      )}
                      {c === IntegrationCategory.Telematics && (
                        <SatelliteDish
                          className="-ms-0.5 me-1.5 opacity-60"
                          size={16}
                          aria-hidden="true"
                        />
                      )}
                      {c === "All" ? "Overview" : getCategoryDisplayName(c)}
                    </TabsTrigger>
                  ))}
                </TabsList>
              </div>
            </Tabs>
          </div>
        </div>
      </section>
      <FormSaveProvider>
        <IntegrationSkeleton />
        {/* <IntegrationGrid search={search} category={category} /> */}
      </FormSaveProvider>
    </>
  );
}
