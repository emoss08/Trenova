/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Loader2, Search, Sparkles } from "lucide-react";
import { useState } from "react";

export function WorkflowTemplateGallery({
  onSelectTemplate,
}: {
  onSelectTemplate?: (templateId: string) => void;
}) {
  const [search, setSearch] = useState("");

  const { data: templates, isLoading } = useQuery(
    queries.workflowTemplate.list(),
  );

  const filteredTemplates = templates?.items?.filter((template) => {
    if (!search) return true;
    return (
      template.name.toLowerCase().includes(search.toLowerCase()) ||
      template.description?.toLowerCase().includes(search.toLowerCase())
    );
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="font-semibold text-xl">Workflow Templates</h2>
          <p className="text-muted-foreground text-sm">
            Start with a pre-built workflow template
          </p>
        </div>
      </div>

      <div className="relative">
        <Search className="absolute top-3 left-3 size-4 text-muted-foreground" />
        <Input
          placeholder="Search templates..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="pl-9"
        />
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="size-8 animate-spin text-muted-foreground" />
        </div>
      ) : filteredTemplates && filteredTemplates.length > 0 ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {filteredTemplates.map((template) => (
            <Card key={template.id} className="transition-shadow hover:shadow-lg">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Sparkles className="size-5 text-primary" />
                  <span className="line-clamp-1">{template.name}</span>
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <p className="line-clamp-3 text-muted-foreground text-sm">
                  {template.description || "No description available"}
                </p>

                <div className="flex items-center justify-between">
                  <div>
                    {template.isSystemTemplate && (
                      <span className="rounded-full bg-primary/10 px-2 py-1 font-medium text-primary text-xs">
                        System
                      </span>
                    )}
                  </div>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => onSelectTemplate?.(template.id)}
                  >
                    Use Template
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : (
        <div className="py-12 text-center">
          <Sparkles className="mx-auto size-12 text-muted-foreground" />
          <h3 className="mt-4 font-medium text-lg">No templates found</h3>
          <p className="mt-2 text-muted-foreground text-sm">
            {search
              ? "Try adjusting your search"
              : "No workflow templates are available"}
          </p>
        </div>
      )}
    </div>
  );
}
