import { BetaTag } from "@/components/ui/beta-tag";
import { Button } from "@/components/ui/button";
import {
  Carousel,
  CarouselContent,
  CarouselItem,
} from "@/components/ui/carousel";
import { Icon } from "@/components/ui/icons";
import { queries } from "@/lib/queries";
import { SuggestionStatus } from "@/lib/schemas/dedicated-lane-schema";
import { cn, formatLocation, pluralize } from "@/lib/utils";
import { Status } from "@/types/common";
import { faDash } from "@fortawesome/pro-solid-svg-icons";
import { useSuspenseQuery } from "@tanstack/react-query";

export function DedicatedLaneSuggestions() {
  const { data, isLoading } = useSuspenseQuery({
    ...queries.dedicatedLaneSuggestion.getSuggestions(),
  });

  if (isLoading) {
    return <div>Loading...</div>;
  }

  return data.results.length > 0 ? (
    <div className="flex flex-col gap-2 bg-background border-border border rounded-md p-2 mb-4 relative">
      <div className="flex justify-between items-center border-b border-border pb-2">
        <div className="flex flex-col items-start leading-tight">
          <div className="flex items-center gap-1">
            <h2 className="text-lg font-semibold">
              Dedicated Lane Suggestions
            </h2>
            <BetaTag />
          </div>
          <p className="text-sm text-muted-foreground">
            A dedicated lane is a lane that is assigned to a specific customer
            for a specific period of time.
          </p>
        </div>
        <div className="flex flex-col items-end">
          <Button size="sm">View All</Button>
          <p className="text-2xs text-muted-foreground">
            {data.results.length} {pluralize("suggestion", data.results.length)}
          </p>
        </div>
      </div>
      <Carousel>
        <CarouselContent className="-ml-4">
          {data.results.map((suggestion) => (
            <CarouselItem className="basis-1/3 pl-4" key={suggestion.id}>
              <div className="flex flex-col gap-2 w-full bg-muted border border-border rounded-md p-2">
                <div className="flex justify-between items-center gap-2 border-b border-border pb-2">
                  <h3 className="text-sm font-semibold">
                    {suggestion.customer?.name}
                  </h3>
                  <LaneSuggestionStatus status={suggestion.status} />
                </div>
                <div className="flex justify-between items-center">
                  <div className="flex flex-col leading-tight">
                    <div className="text-sm font-semibold">
                      {suggestion.originLocation?.name}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {formatLocation(
                        suggestion?.originLocation ?? {
                          name: "Unknown",
                          status: Status.Active,
                          code: "UNKNOWN",
                          addressLine1: "Unknown",
                          city: "Unknown",
                          stateId: "Unknown",
                          postalCode: "Unknown",
                          isGeocoded: false,
                          locationCategoryId: "UNKNOWN",
                        },
                      )}
                    </div>
                  </div>
                  <Icon icon={faDash} className="text-primary" />
                  <div className="flex flex-col leading-tight">
                    <div className="text-sm font-semibold">
                      {suggestion.destinationLocation?.name}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {formatLocation(
                        suggestion?.destinationLocation ?? {
                          name: "Unknown",
                          status: Status.Active,
                          code: "UNKNOWN",
                          addressLine1: "Unknown",
                          city: "Unknown",
                          stateId: "Unknown",
                          postalCode: "Unknown",
                          isGeocoded: false,
                          locationCategoryId: "UNKNOWN",
                        },
                      )}
                    </div>
                  </div>
                </div>
              </div>
            </CarouselItem>
          ))}
        </CarouselContent>
      </Carousel>
    </div>
  ) : null;
}

function LaneSuggestionStatus({ status }: { status: SuggestionStatus }) {
  return (
    <p
      className={cn(
        "text-sm font-semibold uppercase",
        status === SuggestionStatus.Pending &&
          "text-orange-500 dark:text-yellow-500 dark:text-shadow-yellow-500/20 dark:text-shadow-lg",
        status === SuggestionStatus.Accepted &&
          "text-green-500 dark:text-green-500 text-shadow-lg text-shadow-green-500/20 dark:text-shadow-lg",
        status === SuggestionStatus.Rejected &&
          "text-red-500 text-shadow-lg text-shadow-red-500/20 dark:text-shadow-lg",
        status === SuggestionStatus.Expired &&
          "text-red-500 text-shadow-lg text-shadow-red-500/20 dark:text-shadow-lg",
      )}
    >
      {status}
    </p>
  );
}
