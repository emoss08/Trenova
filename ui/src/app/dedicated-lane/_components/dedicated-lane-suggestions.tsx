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
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Icon } from "@/components/ui/icons";
import { EntityRedirectLink } from "@/components/ui/link";
import { type DedicatedLaneSuggestionSchema } from "@/lib/schemas/dedicated-lane-schema";
import { formatLocation, pluralize } from "@/lib/utils";
import { Status } from "@/types/common";
import { faDash, faEllipsis } from "@fortawesome/pro-solid-svg-icons";
import { parseAsBoolean, useQueryState } from "nuqs";
import { AcceptSuggestionDialog } from "./modals/suggestion-accept-modal";
import { RejectSuggestionDialog } from "./modals/suggestion-rejection-modal";

const dialogs = {
  acceptSuggestionDialogOpen: parseAsBoolean.withDefault(false),
  rejectSuggestionDialogOpen: parseAsBoolean.withDefault(false),
};

export function DedicatedLaneSuggestions({
  suggestions,
  isLoading,
}: {
  suggestions: DedicatedLaneSuggestionSchema[];
  isLoading: boolean;
}) {
  if (isLoading) {
    return <div>Loading...</div>;
  }

  return suggestions.length > 0 ? (
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
            {suggestions.length} {pluralize("suggestion", suggestions.length)}
          </p>
        </div>
      </div>
      <Carousel>
        <CarouselContent className="-ml-4 pb-2">
          {suggestions.map((suggestion) => (
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
                  <SuggestionActions suggestion={suggestion} />
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

function SuggestionActions({
  suggestion,
}: {
  suggestion: DedicatedLaneSuggestionSchema;
}) {
  const [acceptSuggestionDialogOpen, setAcceptSuggestionDialogOpen] =
    useQueryState<boolean>(
      "acceptSuggestionDialogOpen",
      dialogs.acceptSuggestionDialogOpen.withOptions({}),
    );

  const [rejectSuggestionDialogOpen, setRejectSuggestionDialogOpen] =
    useQueryState<boolean>(
      "rejectSuggestionDialogOpen",
      dialogs.rejectSuggestionDialogOpen.withOptions({}),
    );

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="xs">
            <Icon icon={faEllipsis} />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuLabel>Actions</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem title="View" description="View this suggestion" />
          <DropdownMenuItem
            color="success"
            title="Accept"
            description="Accept this suggestion"
            onClick={() => setAcceptSuggestionDialogOpen(true)}
          />
          <DropdownMenuItem
            color="danger"
            title="Reject"
            description="Reject this suggestion"
            onClick={() => setRejectSuggestionDialogOpen(true)}
          />
        </DropdownMenuContent>
      </DropdownMenu>
      {acceptSuggestionDialogOpen && (
        <AcceptSuggestionDialog
          open={acceptSuggestionDialogOpen}
          onOpenChange={setAcceptSuggestionDialogOpen}
          suggestion={suggestion}
        />
      )}
      {rejectSuggestionDialogOpen && (
        <RejectSuggestionDialog
          open={rejectSuggestionDialogOpen}
          onOpenChange={setRejectSuggestionDialogOpen}
          suggestion={suggestion}
        />
      )}
    </>
  );
}
