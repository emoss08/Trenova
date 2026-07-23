import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import {
  SHIPMENT_VARIABLES,
  VARIABLE_CATEGORIES,
  AVAILABLE_FUNCTIONS,
} from "@/types/formula-template";
import { BookOpen, Variable, FunctionSquare, ChevronDown } from "lucide-react";

type Tab = "variables" | "functions";

export function FormulaReferencePanel({ className }: { className?: string }) {
  const [activeTab, setActiveTab] = useState<Tab>("variables");
  const [isCollapsed, setIsCollapsed] = useState(false);

  return (
    <Card className={cn("overflow-hidden", className)}>
      <CardHeader
        className="flex cursor-pointer flex-row items-center justify-between border-b py-3"
        onClick={() => setIsCollapsed(!isCollapsed)}
      >
        <div className="flex items-center gap-2">
          <div className="flex size-8 items-center justify-center rounded-lg bg-primary/10">
            <BookOpen className="size-4 text-primary" />
          </div>
          <div>
            <CardTitle className="text-sm font-medium">Reference</CardTitle>
            <p className="text-xs text-muted-foreground">
              Available variables and functions
            </p>
          </div>
        </div>
        <ChevronDown
          className={cn(
            "size-4 text-muted-foreground transition-transform",
            isCollapsed && "-rotate-90",
          )}
        />
      </CardHeader>

      {!isCollapsed && (
        <CardContent className="p-0">
          <div className="flex border-b">
            <button
              type="button"
              onClick={() => setActiveTab("variables")}
              className={cn(
                "flex flex-1 items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium transition-colors",
                activeTab === "variables"
                  ? "border-b-2 border-primary text-primary"
                  : "text-muted-foreground hover:text-foreground",
              )}
            >
              <Variable className="size-3.5" />
              Variables
            </button>
            <button
              type="button"
              onClick={() => setActiveTab("functions")}
              className={cn(
                "flex flex-1 items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium transition-colors",
                activeTab === "functions"
                  ? "border-b-2 border-primary text-primary"
                  : "text-muted-foreground hover:text-foreground",
              )}
            >
              <FunctionSquare className="size-3.5" />
              Functions
            </button>
          </div>

          <div className="max-h-[300px] overflow-y-auto p-2">
            {activeTab === "variables" && (
              <div className="space-y-3">
                {VARIABLE_CATEGORIES.map((category) => {
                  const categoryVars = SHIPMENT_VARIABLES.filter(
                    (v) => v.category === category.id,
                  );
                  if (categoryVars.length === 0) return null;

                  return (
                    <div key={category.id}>
                      <h4 className="mb-1 px-2 text-[10px] font-semibold tracking-wide text-muted-foreground uppercase">
                        {category.label}
                      </h4>
                      <div className="space-y-1">
                        {categoryVars.map((variable) => (
                          <div
                            key={variable.name}
                            className="group flex items-start gap-3 rounded-md p-2 transition-colors hover:bg-muted/50"
                          >
                            <code className="mt-0.5 shrink-0 rounded bg-primary/10 px-1.5 py-0.5 font-mono text-xs text-primary">
                              {variable.name}
                            </code>
                            <div className="min-w-0 flex-1">
                              <p className="text-xs text-muted-foreground">
                                {variable.description}
                              </p>
                            </div>
                            <span className="shrink-0 rounded bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
                              {variable.type}
                            </span>
                          </div>
                        ))}
                      </div>
                      {category.id === "computed" && (
                        <p className="mt-2 px-2 text-[10px] text-muted-foreground italic">
                          Commodity data (weight, pieces, hazmat) is available
                          through computed rollup variables. Direct commodity
                          iteration is not supported in formulas.
                        </p>
                      )}
                    </div>
                  );
                })}
              </div>
            )}

            {activeTab === "functions" && (
              <div className="space-y-1">
                {AVAILABLE_FUNCTIONS.map((func) => (
                  <div
                    key={func.name}
                    className="group flex items-start gap-3 rounded-md p-2 transition-colors hover:bg-muted/50"
                  >
                    <code className="mt-0.5 shrink-0 rounded bg-chart-2/10 px-1.5 py-0.5 font-mono text-xs text-chart-2">
                      {func.signature}
                    </code>
                    <div className="min-w-0 flex-1">
                      <p className="text-xs text-muted-foreground">
                        {func.description}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </CardContent>
      )}
    </Card>
  );
}
