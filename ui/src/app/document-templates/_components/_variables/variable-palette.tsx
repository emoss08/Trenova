import { documentTemplateEditorParser } from "@/app/workers/_components/pto/use-document-template-state";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import {
  Building2,
  Calendar,
  CircleDollarSign,
  Copy,
  FileText,
  Hash,
  MapPin,
  Package,
  Search,
  Truck,
  User,
  Weight,
} from "lucide-react";
import { useQueryStates } from "nuqs";
import { useMemo, useState } from "react";

interface TemplateVariable {
  name: string;
  description: string;
  example: string;
  syntax: string;
}

interface VariableCategory {
  id: string;
  name: string;
  icon: React.ElementType;
  color: string;
  variables: TemplateVariable[];
}

const variableCategories: VariableCategory[] = [
  {
    id: "document",
    name: "Document",
    icon: FileText,
    color: "bg-blue-500",
    variables: [
      {
        name: "DocumentNumber",
        description: "Unique document identifier",
        example: "INV-2024-0001",
        syntax: "{{ .DocumentNumber }}",
      },
      {
        name: "DocumentDate",
        description: "Date the document was created",
        example: "January 15, 2024",
        syntax: "{{ formatDate .DocumentDate }}",
      },
      {
        name: "DueDate",
        description: "Payment or action due date",
        example: "February 15, 2024",
        syntax: "{{ formatDate .DueDate }}",
      },
      {
        name: "PageNumber",
        description: "Current page number",
        example: "1",
        syntax: "{{ .PageNumber }}",
      },
      {
        name: "TotalPages",
        description: "Total number of pages",
        example: "3",
        syntax: "{{ .TotalPages }}",
      },
    ],
  },
  {
    id: "company",
    name: "Company",
    icon: Building2,
    color: "bg-purple-500",
    variables: [
      {
        name: "CompanyName",
        description: "Your company name",
        example: "Trenova Logistics Inc.",
        syntax: "{{ .CompanyName }}",
      },
      {
        name: "CompanyLogo",
        description: "URL to company logo",
        example: "https://...",
        syntax: "{{ .CompanyLogo }}",
      },
      {
        name: "CompanyAddress",
        description: "Full company address",
        example: "123 Main St, City, ST 12345",
        syntax: "{{ .CompanyAddress }}",
      },
      {
        name: "CompanyPhone",
        description: "Company phone number",
        example: "(555) 123-4567",
        syntax: "{{ .CompanyPhone }}",
      },
      {
        name: "CompanyEmail",
        description: "Company email address",
        example: "info@company.com",
        syntax: "{{ .CompanyEmail }}",
      },
    ],
  },
  {
    id: "customer",
    name: "Customer",
    icon: User,
    color: "bg-green-500",
    variables: [
      {
        name: "CustomerName",
        description: "Customer's full name or company",
        example: "Acme Corporation",
        syntax: "{{ .CustomerName }}",
      },
      {
        name: "CustomerAddress",
        description: "Customer's address",
        example: "456 Oak Ave, Town, ST 67890",
        syntax: "{{ .CustomerAddress }}",
      },
      {
        name: "CustomerEmail",
        description: "Customer's email",
        example: "contact@acme.com",
        syntax: "{{ .CustomerEmail }}",
      },
      {
        name: "CustomerPhone",
        description: "Customer's phone",
        example: "(555) 987-6543",
        syntax: "{{ .CustomerPhone }}",
      },
    ],
  },
  {
    id: "shipment",
    name: "Shipment",
    icon: Truck,
    color: "bg-orange-500",
    variables: [
      {
        name: "ShipmentID",
        description: "Shipment identifier",
        example: "SHP-2024-1234",
        syntax: "{{ .ShipmentID }}",
      },
      {
        name: "ProNumber",
        description: "PRO tracking number",
        example: "PRO123456789",
        syntax: "{{ .ProNumber }}",
      },
      {
        name: "BOLNumber",
        description: "Bill of Lading number",
        example: "BOL-2024-5678",
        syntax: "{{ .BOLNumber }}",
      },
      {
        name: "PickupDate",
        description: "Scheduled pickup date",
        example: "January 20, 2024",
        syntax: "{{ formatDate .PickupDate }}",
      },
      {
        name: "DeliveryDate",
        description: "Scheduled delivery date",
        example: "January 22, 2024",
        syntax: "{{ formatDate .DeliveryDate }}",
      },
    ],
  },
  {
    id: "location",
    name: "Location",
    icon: MapPin,
    color: "bg-red-500",
    variables: [
      {
        name: "OriginName",
        description: "Origin location name",
        example: "Chicago Warehouse",
        syntax: "{{ .OriginName }}",
      },
      {
        name: "OriginAddress",
        description: "Full origin address",
        example: "789 Industrial Blvd, Chicago, IL 60601",
        syntax: "{{ .OriginAddress }}",
      },
      {
        name: "DestinationName",
        description: "Destination location name",
        example: "New York Distribution Center",
        syntax: "{{ .DestinationName }}",
      },
      {
        name: "DestinationAddress",
        description: "Full destination address",
        example: "321 Commerce St, New York, NY 10001",
        syntax: "{{ .DestinationAddress }}",
      },
    ],
  },
  {
    id: "financial",
    name: "Financial",
    icon: CircleDollarSign,
    color: "bg-green-500",
    variables: [
      {
        name: "Subtotal",
        description: "Sum before taxes/fees",
        example: "$1,250.00",
        syntax: "{{ formatCurrency .Subtotal }}",
      },
      {
        name: "TaxAmount",
        description: "Total tax amount",
        example: "$100.00",
        syntax: "{{ formatCurrency .TaxAmount }}",
      },
      {
        name: "TotalAmount",
        description: "Grand total",
        example: "$1,350.00",
        syntax: "{{ formatCurrency .TotalAmount }}",
      },
      {
        name: "AmountPaid",
        description: "Amount already paid",
        example: "$500.00",
        syntax: "{{ formatCurrency .AmountPaid }}",
      },
      {
        name: "BalanceDue",
        description: "Remaining balance",
        example: "$850.00",
        syntax: "{{ formatCurrency .BalanceDue }}",
      },
    ],
  },
  {
    id: "cargo",
    name: "Cargo",
    icon: Package,
    color: "bg-amber-500",
    variables: [
      {
        name: "TotalPieces",
        description: "Total number of pieces",
        example: "24",
        syntax: "{{ .TotalPieces }}",
      },
      {
        name: "TotalWeight",
        description: "Total shipment weight",
        example: "2,500 lbs",
        syntax: "{{ formatWeight .TotalWeight }}",
      },
      {
        name: "Commodity",
        description: "Type of goods",
        example: "Electronics",
        syntax: "{{ .Commodity }}",
      },
      {
        name: "HazmatClass",
        description: "Hazmat classification if applicable",
        example: "Class 3",
        syntax: "{{ .HazmatClass }}",
      },
    ],
  },
];

const loopTemplates = [
  {
    name: "Line Items",
    description: "Iterate over invoice line items",
    syntax: `{{ range .LineItems }}
  <tr>
    <td>{{ .Description }}</td>
    <td>{{ .Quantity }}</td>
    <td>{{ formatCurrency .UnitPrice }}</td>
    <td>{{ formatCurrency .Total }}</td>
  </tr>
{{ end }}`,
  },
  {
    name: "Stops",
    description: "Iterate over shipment stops",
    syntax: `{{ range .Stops }}
  <div class="stop">
    <strong>{{ .Type }}</strong>: {{ .LocationName }}
    <br>{{ .Address }}
  </div>
{{ end }}`,
  },
  {
    name: "Commodities",
    description: "Iterate over shipment commodities",
    syntax: `{{ range .Commodities }}
  <tr>
    <td>{{ .Description }}</td>
    <td>{{ .Pieces }}</td>
    <td>{{ formatWeight .Weight }}</td>
  </tr>
{{ end }}`,
  },
];

const helperFunctions = [
  {
    name: "formatDate",
    description: "Format a date value",
    example: "January 15, 2024",
    syntax: "{{ formatDate .Date }}",
  },
  {
    name: "formatCurrency",
    description: "Format a number as currency",
    example: "$1,234.56",
    syntax: "{{ formatCurrency .Amount }}",
  },
  {
    name: "formatWeight",
    description: "Format weight with units",
    example: "2,500 lbs",
    syntax: "{{ formatWeight .Weight }}",
  },
  {
    name: "upper",
    description: "Convert text to uppercase",
    example: "HELLO WORLD",
    syntax: "{{ upper .Text }}",
  },
  {
    name: "lower",
    description: "Convert text to lowercase",
    example: "hello world",
    syntax: "{{ lower .Text }}",
  },
];

export function VariablePalette({
  onInsert,
  className,
}: {
  onInsert: (syntax: string) => void;
  className?: string;
}) {
  const [searchParams, setSearchParams] = useQueryStates(
    documentTemplateEditorParser,
  );
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);
  const [copiedSyntax, setCopiedSyntax] = useState<string | null>(null);

  const filteredCategories = useMemo(() => {
    if (!searchParams.variableSearchQuery) return variableCategories;

    const query = searchParams.variableSearchQuery.toLowerCase();
    return variableCategories
      .map((category) => ({
        ...category,
        variables: category.variables.filter(
          (v) =>
            v.name.toLowerCase().includes(query) ||
            v.description.toLowerCase().includes(query),
        ),
      }))
      .filter((category) => category.variables.length > 0);
  }, [searchParams.variableSearchQuery]);

  const handleCopy = async (syntax: string) => {
    await navigator.clipboard.writeText(syntax);
    setCopiedSyntax(syntax);
    setTimeout(() => setCopiedSyntax(null), 2000);
  };

  const handleInsertAndCopy = (syntax: string) => {
    onInsert(syntax);
    handleCopy(syntax);
  };

  return (
    <div className={cn("flex flex-col min-h-[calc(100vh-20rem)]", className)}>
      <div className="shrink-0 border-b border-border p-3">
        <div className="relative">
          <Search className="absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search variables..."
            value={searchParams.variableSearchQuery}
            onChange={(e) =>
              setSearchParams({ variableSearchQuery: e.target.value })
            }
            className="h-8 pl-8 text-sm"
          />
        </div>
      </div>

      {!searchParams.variableSearchQuery && (
        <div className="flex flex-wrap gap-1.5 p-2">
          {variableCategories.map((category) => {
            const Icon = category.icon;
            const isSelected = selectedCategory === category.id;

            return (
              <TooltipProvider key={category.id}>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <button
                      type="button"
                      onClick={() =>
                        setSelectedCategory(isSelected ? null : category.id)
                      }
                      className={cn(
                        "flex size-9 items-center justify-center rounded-lg transition-all",
                        isSelected
                          ? `${category.color} text-white shadow-md`
                          : "bg-muted/50 text-muted-foreground hover:bg-muted hover:text-foreground",
                      )}
                    >
                      <Icon className="size-4" />
                    </button>
                  </TooltipTrigger>
                  <TooltipContent side="bottom">{category.name}</TooltipContent>
                </Tooltip>
              </TooltipProvider>
            );
          })}
        </div>
      )}
      <ScrollArea className="flex max-h-[calc(100vh-25rem)] flex-col gap-2 p-4">
        <div className="flex flex-col gap-2">
          {(selectedCategory
            ? filteredCategories.filter((c) => c.id === selectedCategory)
            : filteredCategories
          ).map((category) => {
            const Icon = category.icon;

            return (
              <div className="flex flex-col gap-2" key={category.id}>
                <div className="flex items-center gap-2">
                  <div
                    className={cn(
                      "flex size-6 items-center justify-center rounded-md bg-gradient-to-br text-white",
                      category.color,
                    )}
                  >
                    <Icon className="size-3.5" />
                  </div>
                  <span className="text-sm font-medium">{category.name}</span>
                </div>
                <div className="space-y-1">
                  {category.variables.map((variable) => (
                    <button
                      key={variable.name}
                      type="button"
                      onClick={() => handleInsertAndCopy(variable.syntax)}
                      className="group flex w-full items-start gap-2 rounded-lg border border-transparent bg-muted/30 p-2 text-left transition-all hover:border-primary/20 hover:bg-muted/50"
                    >
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <code className="rounded bg-primary/10 px-1.5 py-0.5 text-xs font-medium text-primary">
                            {variable.name}
                          </code>
                          {copiedSyntax === variable.syntax && (
                            <span className="text-2xs text-green-600">
                              Copied!
                            </span>
                          )}
                        </div>
                        <p className="mt-0.5 line-clamp-1 text-xs text-muted-foreground">
                          {variable.description}
                        </p>
                        <p className="mt-0.5 text-2xs text-muted-foreground/70">
                          e.g. {variable.example}
                        </p>
                      </div>
                      <Copy className="size-3.5 shrink-0 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100" />
                    </button>
                  ))}
                </div>
              </div>
            );
          })}
          {!selectedCategory && !searchParams.variableSearchQuery && (
            <div className="border-t border-border pt-2">
              <div className="mb-2 flex items-center gap-2">
                <div className="flex size-6 items-center justify-center rounded-md bg-gradient-to-br from-indigo-500 to-indigo-600 text-white">
                  <Hash className="size-3.5" />
                </div>
                <span className="text-sm font-medium">Loops</span>
              </div>
              <div className="space-y-1">
                {loopTemplates.map((loop) => (
                  <button
                    key={loop.name}
                    type="button"
                    onClick={() => handleInsertAndCopy(loop.syntax)}
                    className="group flex w-full items-start gap-2 rounded-lg border border-transparent bg-muted/30 p-2 text-left transition-all hover:border-primary/20 hover:bg-muted/50"
                  >
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <code className="rounded bg-indigo-500/10 px-1.5 py-0.5 text-xs font-medium text-indigo-600 dark:text-indigo-400">
                          {loop.name}
                        </code>
                      </div>
                      <p className="mt-0.5 text-xs text-muted-foreground">
                        {loop.description}
                      </p>
                    </div>
                    <Copy className="size-3.5 shrink-0 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100" />
                  </button>
                ))}
              </div>
            </div>
          )}
          {!selectedCategory && !searchParams.variableSearchQuery && (
            <div className="border-t border-border pt-2">
              <div className="mb-2 flex items-center gap-2">
                <div className="flex size-6 items-center justify-center rounded-md bg-gradient-to-br from-pink-500 to-pink-600 text-white">
                  <Calendar className="size-3.5" />
                </div>
                <span className="text-sm font-medium">Helper Functions</span>
              </div>
              <div className="space-y-1">
                {helperFunctions.map((fn) => (
                  <button
                    key={fn.name}
                    type="button"
                    onClick={() => handleInsertAndCopy(fn.syntax)}
                    className="group flex w-full items-start gap-2 rounded-lg border border-transparent bg-muted/30 p-2 text-left transition-all hover:border-primary/20 hover:bg-muted/50"
                  >
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <code className="rounded bg-pink-500/10 px-1.5 py-0.5 text-xs font-medium text-pink-600 dark:text-pink-400">
                          {fn.name}()
                        </code>
                      </div>
                      <p className="mt-0.5 text-xs text-muted-foreground">
                        {fn.description}
                      </p>
                      <p className="mt-0.5 text-2xs text-muted-foreground/70">
                        e.g. {fn.example}
                      </p>
                    </div>
                    <Copy className="size-3.5 shrink-0 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100" />
                  </button>
                ))}
              </div>
            </div>
          )}

          {!selectedCategory && !searchParams.variableSearchQuery && (
            <div className="border-t border-border pt-4">
              <div className="mb-2 flex items-center gap-2">
                <div className="flex size-6 items-center justify-center rounded-md bg-gradient-to-br from-cyan-500 to-cyan-600 text-white">
                  <Weight className="size-3.5" />
                </div>
                <span className="text-sm font-medium">Conditionals</span>
              </div>
              <div className="space-y-1">
                <button
                  type="button"
                  onClick={() =>
                    handleInsertAndCopy("{{ if .HasValue }}...{{ end }}")
                  }
                  className="group flex w-full items-start gap-2 rounded-lg border border-transparent bg-muted/30 p-2 text-left transition-all hover:border-primary/20 hover:bg-muted/50"
                >
                  <div className="min-w-0 flex-1">
                    <code className="rounded bg-cyan-500/10 px-1.5 py-0.5 text-xs font-medium text-cyan-600 dark:text-cyan-400">
                      if...end
                    </code>
                    <p className="mt-0.5 text-xs text-muted-foreground">
                      Conditionally show content
                    </p>
                  </div>
                  <Copy className="size-3.5 shrink-0 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100" />
                </button>
                <button
                  type="button"
                  onClick={() =>
                    handleInsertAndCopy(
                      "{{ if .Value }}...{{ else }}...{{ end }}",
                    )
                  }
                  className="group flex w-full items-start gap-2 rounded-lg border border-transparent bg-muted/30 p-2 text-left transition-all hover:border-primary/20 hover:bg-muted/50"
                >
                  <div className="min-w-0 flex-1">
                    <code className="rounded bg-cyan-500/10 px-1.5 py-0.5 text-xs font-medium text-cyan-600 dark:text-cyan-400">
                      if...else...end
                    </code>
                    <p className="mt-0.5 text-xs text-muted-foreground">
                      Show different content based on condition
                    </p>
                  </div>
                  <Copy className="size-3.5 shrink-0 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100" />
                </button>
              </div>
            </div>
          )}
        </div>
      </ScrollArea>
    </div>
  );
}
