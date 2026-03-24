import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { ColorSchemeId } from "@nivo/colors";

export type ApprovedPTOColorScheme = "nivo" | "greys";

export const APPROVED_PTO_COLOR_SCHEMES = [
  { value: "nivo", label: "Light" },
  { value: "greys", label: "Dark" },
] as const satisfies ReadonlyArray<{ value: ApprovedPTOColorScheme; label: string }>;

export function isApprovedPTOColorScheme(value: string): value is ColorSchemeId {
  return APPROVED_PTO_COLOR_SCHEMES.some((scheme) => scheme.value === value);
}

export function ApprovedChartOptions({
  colorScheme,
  setColorScheme,
}: {
  colorScheme: ApprovedPTOColorScheme;
  setColorScheme: (colorScheme: ApprovedPTOColorScheme) => void;
}) {
  return (
    <ChartOptionsOuter>
      <ChartOptionsInner>
        <p className="text-sm text-muted-foreground">Color Scheme:</p>
        <Select
          items={APPROVED_PTO_COLOR_SCHEMES}
          value={colorScheme}
          onValueChange={(v) => setColorScheme(v as ApprovedPTOColorScheme)}
        >
          <SelectTrigger className="h-8 w-37.5 text-xs">
            <SelectValue placeholder="Color" />
          </SelectTrigger>
          <SelectContent>
            {APPROVED_PTO_COLOR_SCHEMES.map((scheme) => (
              <SelectItem key={scheme.value} value={scheme.value} className="text-xs">
                {scheme.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </ChartOptionsInner>
    </ChartOptionsOuter>
  );
}

function ChartOptionsOuter({ children }: { children: React.ReactNode }) {
  return <div className="absolute top-0 right-0 z-10">{children}</div>;
}

function ChartOptionsInner({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center gap-2">{children}</div>;
}
