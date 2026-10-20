import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  DeskFuelCardCardDocument,
  DeskFuelCardPurchaseFieldsFragmentDoc,
  DeskFuelPurchaseCardDocument,
  FuelCardFieldsFragmentDoc,
  FuelPurchaseFieldsFragmentDoc,
  type DeskFuelCardCardQuery,
  type DeskFuelCardPurchaseFieldsFragment,
  type DeskFuelPurchaseCardQuery,
  type FuelCardFieldsFragment,
  type FuelPurchaseFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

type RawPurchaseCard = DeskFuelPurchaseCardQuery["fuelPurchaseDeskCard"];
type RawCardCard = DeskFuelCardCardQuery["fuelCardDeskCard"];
type RawTractor = NonNullable<RawPurchaseCard["purchase"]["tractor"]>;

export type DeskFuelCard = FuelCardFieldsFragment;
export type DeskFuelTractor = RawTractor;

/** A purchase as the Desk card draws it: the row's fields, its tractor's make and model, and its card whole. */
export type DeskFuelPurchase = Omit<FuelPurchaseFieldsFragment, "tractor" | "fuelCard"> & {
  tractor: RawTractor | null;
  fuelCard: DeskFuelCard | null;
};

export type DeskFuelPurchaseCard = Omit<RawPurchaseCard, "purchase"> & {
  purchase: DeskFuelPurchase;
};

export type DeskFuelCardPurchase = DeskFuelCardPurchaseFieldsFragment;
export type DeskFuelHeldRow = RawCardCard["held"][number];
export type DeskFuelHeldBatch = RawCardCard["heldBatches"][number];
export type DeskFuelSpend = RawCardCard["month"][number];
export type DeskFuelHeldSpend = RawCardCard["heldSpend"][number];
export type DeskFuelJurisdictionGallons = RawCardCard["quarterByJurisdiction"][number];
export type DeskFuelFeed = NonNullable<RawCardCard["feed"]>;

/** A fuel card as the Desk card draws it: the card whole, with its tractor's make and model. */
export type DeskFuelCardRecord = Omit<FuelCardFieldsFragment, "assignedTractor"> & {
  assignedTractor: RawTractor | null;
};

export type DeskFuelCardCard = Omit<RawCardCard, "card" | "recent" | "successors"> & {
  card: DeskFuelCardRecord;
  recent: DeskFuelCardPurchase[];
  successors: DeskFuelCard[];
};

export async function fetchDeskFuelPurchase(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<DeskFuelPurchaseCard> {
  const data = await requestGraphQL({
    document: DeskFuelPurchaseCardDocument,
    operationName: "DeskFuelPurchaseCard",
    variables: { id },
    signal: options?.signal,
  });
  const card = data.fuelPurchaseDeskCard;
  const fields = getFragmentData(FuelPurchaseFieldsFragmentDoc, card.purchase);

  return {
    ...card,
    purchase: {
      ...fields,
      tractor: card.purchase.tractor,
      fuelCard: card.purchase.fuelCard
        ? getFragmentData(FuelCardFieldsFragmentDoc, card.purchase.fuelCard)
        : null,
    },
  };
}

export async function fetchDeskFuelCard(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<DeskFuelCardCard> {
  const data = await requestGraphQL({
    document: DeskFuelCardCardDocument,
    operationName: "DeskFuelCardCard",
    variables: { id },
    signal: options?.signal,
  });
  const card = data.fuelCardDeskCard;
  const fields = getFragmentData(FuelCardFieldsFragmentDoc, card.card);

  return {
    ...card,
    card: { ...fields, assignedTractor: card.card.assignedTractor },
    recent: card.recent.map((row) => getFragmentData(DeskFuelCardPurchaseFieldsFragmentDoc, row)),
    successors: card.successors.map((row) => getFragmentData(FuelCardFieldsFragmentDoc, row)),
  };
}
