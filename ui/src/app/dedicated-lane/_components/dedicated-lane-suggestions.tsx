import { BetaTag } from "@/components/ui/beta-tag";
import { Button } from "@/components/ui/button";
import {
  Carousel,
  CarouselContent,
  CarouselItem,
} from "@/components/ui/carousel";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Icon } from "@/components/ui/icons";
import { EntityRedirectLink } from "@/components/ui/link";
import { queries } from "@/lib/queries";
import { formatLocation, pluralize } from "@/lib/utils";
import { Status } from "@/types/common";
import { faDash, faEllipsis } from "@fortawesome/pro-solid-svg-icons";
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
        <CarouselContent className="-ml-4 pb-2">
          {data.results.map((suggestion) => (
            <CarouselItem className="basis-1/3 pl-4" key={suggestion.id}>
              <div className="flex flex-col gap-2 w-full bg-muted border border-border rounded-md p-2">
                <div className="flex justify-between items-center gap-2 border-b border-border pb-2">
                  <EntityRedirectLink
                    baseUrl="/billing/configurations/customers"
                    entityId={suggestion.customerId}
                    className="text-sm !font-semibold"
                    modelOpen
                  >
                    {suggestion.customer?.name}
                  </EntityRedirectLink>
                  <SuggestionActions />
                </div>
                <div className="flex justify-between items-center">
                  <div className="flex flex-col leading-tight">
                    <div className="text-sm font-semibold">
                      <EntityRedirectLink
                        baseUrl="/dispatch/configurations/locations"
                        entityId={suggestion.originLocationId}
                        className="text-xs !font-semibold max-w-[100px] truncate"
                        modelOpen
                      >
                        {suggestion.originLocation?.name}
                      </EntityRedirectLink>
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
                      <EntityRedirectLink
                        baseUrl="/dispatch/configurations/locations"
                        entityId={suggestion.destinationLocationId}
                        className="text-xs !font-semibold max-w-[100px] truncate"
                        modelOpen
                      >
                        {suggestion.destinationLocation?.name}
                      </EntityRedirectLink>
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

function SuggestionActions() {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="xs">
          <Icon icon={faEllipsis} />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DropdownMenuItem title="Accept" description="Accept this suggestion" />
        <DropdownMenuItem title="Reject" description="Reject this suggestion" />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
