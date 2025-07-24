/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
