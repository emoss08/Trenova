import {
  ActivateFiscalPeriodDocument,
  CloseFiscalPeriodDocument,
  FiscalPeriodCloseBlockersDocument,
  LockFiscalPeriodDocument,
  ReopenFiscalPeriodDocument,
  UnlockFiscalPeriodDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { FiscalPeriod, FiscalPeriodCloseCheck } from "@/types/fiscal-period";

type RequestOptions = { signal?: AbortSignal };

export class FiscalPeriodService {
  async activate(id: FiscalPeriod["id"]): Promise<FiscalPeriod> {
    const response = await requestGraphQL({
      document: ActivateFiscalPeriodDocument,
      operationName: "ActivateFiscalPeriod",
      variables: { id: id ?? "" },
    });
    return response.activateFiscalPeriod as FiscalPeriod;
  }

  async close(id: FiscalPeriod["id"]): Promise<FiscalPeriod> {
    const response = await requestGraphQL({
      document: CloseFiscalPeriodDocument,
      operationName: "CloseFiscalPeriod",
      variables: { id: id ?? "" },
    });
    return response.closeFiscalPeriod as FiscalPeriod;
  }

  async closeBlockers(
    id: FiscalPeriod["id"],
    { signal }: RequestOptions = {},
  ): Promise<FiscalPeriodCloseCheck> {
    const response = await requestGraphQL({
      document: FiscalPeriodCloseBlockersDocument,
      operationName: "FiscalPeriodCloseBlockers",
      variables: { id: id ?? "" },
      signal,
    });
    return response.fiscalPeriodCloseBlockers;
  }

  async reopen(id: FiscalPeriod["id"], reopenReason: string): Promise<FiscalPeriod> {
    const response = await requestGraphQL({
      document: ReopenFiscalPeriodDocument,
      operationName: "ReopenFiscalPeriod",
      variables: { id: id ?? "", reopenReason },
    });
    return response.reopenFiscalPeriod as FiscalPeriod;
  }

  async lock(id: FiscalPeriod["id"]): Promise<FiscalPeriod> {
    const response = await requestGraphQL({
      document: LockFiscalPeriodDocument,
      operationName: "LockFiscalPeriod",
      variables: { id: id ?? "" },
    });
    return response.lockFiscalPeriod as FiscalPeriod;
  }

  async unlock(id: FiscalPeriod["id"]): Promise<FiscalPeriod> {
    const response = await requestGraphQL({
      document: UnlockFiscalPeriodDocument,
      operationName: "UnlockFiscalPeriod",
      variables: { id: id ?? "" },
    });
    return response.unlockFiscalPeriod as FiscalPeriod;
  }
}
