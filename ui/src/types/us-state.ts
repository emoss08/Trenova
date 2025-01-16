import type { BaseModel } from "./common";

export interface UsState extends BaseModel {
  name: string;
  abbreviation: string;
  countryName: string;
  countryIso3: string;
}
