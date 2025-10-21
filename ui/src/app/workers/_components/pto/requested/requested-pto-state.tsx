import { Icon } from "@/components/ui/icons";
import LetterGlitch from "@/components/ui/letter-glitch";
import { Skeleton } from "@/components/ui/skeleton";
import { faExclamationTriangle } from "@fortawesome/pro-solid-svg-icons";

export function RequestedPTOOverviewSkeleton() {
  return (
    <div className="flex flex-col gap-1 overflow-y-hidden border border-border rounded-md p-3 size-full">
      {Array.from({ length: 10 }).map((_, index) => (
        <div
          key={index}
          className="flex items-center justify-center border border-border rounded-md"
        >
          <Skeleton className="h-[70px] w-full" />
        </div>
      ))}
    </div>
  );
}
export function RequestedPTOEmptyState() {
  return (
    <div className="flex flex-col items-center size-full justify-center overflow-hidden border border-border rounded-md">
      <div className="relative size-full">
        <LetterGlitch
          glitchColors={["#9c9c9c", "#696969", "#424242"]}
          glitchSpeed={50}
          centerVignette={true}
          outerVignette={false}
          smooth={true}
          className="size-full"
          canvasClassName="size-full"
        />
        <div className="absolute inset-0 flex flex-col gap-1 items-center justify-center pointer-events-none">
          <p className="text-sm/none px-1 py-0.5 text-center font-medium uppercase select-none font-table dark:text-neutral-900 bg-amber-300 text-amber-950 dark:bg-amber-400">
            No data available
          </p>
          <p className="text-sm/none px-1 py-0.5 text-center font-medium uppercase select-none font-table dark:text-neutral-900 bg-neutral-900 text-white dark:bg-neutral-500">
            Try adjusting your filters or search query
          </p>
        </div>
      </div>
    </div>
  );
}

export function RequestedPTOErrorState() {
  return (
    <div className="flex flex-col items-center size-full justify-center overflow-hidden border border-border rounded-md gap-1 p-3">
      <Icon
        icon={faExclamationTriangle}
        className="text-red-500 size-5 mt-0.5"
      />
      <div className="flex flex-col text-center items-center">
        <p className="font-medium text-red-500">Error loading PTO requests</p>
        <p className="text-xs text-muted-foreground mt-1">
          Looks like we hit a snag. Please try again later.
        </p>
      </div>
    </div>
  );
}
