import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ColorSchemeId, colorSchemes } from "@nivo/colors";

export function ApprovedChartOptions({
  colorScheme,
  setColorScheme,
}: {
  colorScheme: ColorSchemeId;
  setColorScheme: (colorScheme: ColorSchemeId) => void;
}) {
  return (
    <ChartOptionsOuter>
      <ChartOptionsInner>
        <p className="text-xs font-medium">Color Scheme:</p>
        <Select
          value={colorScheme}
          onValueChange={(v) => setColorScheme(v as ColorSchemeId)}
        >
          <SelectTrigger className="w-[120px] h-8 text-xs">
            <SelectValue placeholder="Color" />
          </SelectTrigger>
          <SelectContent className="min-w-[170px]">
            {Object.keys(colorSchemes).map((scheme) => (
              <SelectItem key={scheme} value={scheme} className="text-xs">
                {scheme}
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
