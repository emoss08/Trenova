type LocationDetails = {
  name: string;
  addressLine1: string;
  addressLine2: string;
  city: string;
  state: string;
  stateId: string;
  postalCode: string;
  longitude?: number;
  latitude?: number;
  placeId?: string;
  types: string[];
};

export type AutoCompleteLocationResult = {
  details: LocationDetails[];
  count: number;
};

export type CheckAPIKeyResponse = {
  valid: boolean;
};
