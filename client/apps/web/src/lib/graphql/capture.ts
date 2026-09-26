import { getFragmentData, type FragmentType } from "@trenova/graphql/fragment-data";
import {
  ApproveCaptureDevicePairingDocument,
  AvailableCaptureProfilesDocument,
  CancelCaptureRequestDocument,
  CaptureBatchDetailFieldsFragmentDoc,
  CaptureBatchDocument,
  CaptureBatchRowFieldsFragmentDoc,
  CaptureBatchesDocument,
  CaptureDeviceFieldsFragmentDoc,
  CaptureDeviceFleetFieldsFragmentDoc,
  CaptureDevicePairingDocument,
  CaptureDevicesDocument,
  CaptureItemFieldsFragmentDoc,
  CaptureProfileFieldsFragmentDoc,
  CaptureProfilesDocument,
  CaptureRequestFieldsFragmentDoc,
  CaptureRequestsForTargetDocument,
  CreateCaptureCoverSheetsDocument,
  CreateCaptureProfileDocument,
  CreateCaptureRequestDocument,
  DeleteCaptureProfileDocument,
  DenyCaptureDevicePairingDocument,
  DiscardCaptureBatchDocument,
  DiscardCaptureItemDocument,
  EditCaptureItemsDocument,
  FileCaptureItemDocument,
  FileCaptureItemsDocument,
  MyCaptureDevicesDocument,
  RevokeCaptureDeviceDocument,
  RevokeMyCaptureDeviceDocument,
  UpdateCaptureProfileDocument,
  type CaptureBatchDetailFieldsFragment,
  type CaptureBatchRowFieldsFragment,
  type CaptureBatchSort,
  type CaptureBatchStatus,
  type CaptureCoverSheetInput,
  type CaptureDeviceFieldsFragment,
  type CaptureDeviceFleetFieldsFragment,
  type CaptureDevicePairingQuery,
  type CaptureDeviceStatus,
  type CaptureItemFieldsFragment,
  type CaptureItemStatus,
  type CapturePageFieldsFragment,
  type CapturePixelType,
  type CaptureProfileFieldsFragment,
  type CaptureProfileInput,
  type CaptureProfileStatus,
  type CaptureRecordRefFieldsFragment,
  type CaptureRequestFieldsFragment,
  type CaptureRequestMode,
  type CaptureRequestStatus,
  type CaptureSeparatorStrategy,
  type CaptureSource,
  type CaptureSourceInfoFieldsFragment,
  type CaptureSuggestionSource,
  type CreateCaptureCoverSheetsMutation,
  type CreateCaptureRequestInput,
  type EditCaptureItemsInput,
  type FileCaptureItemInput,
  type FileCaptureItemsEntryInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { UnmaskFragments } from "@trenova/shared/types/graphql-connection";

/*
 * Unmasked all the way down: a batch's detail fragment spreads its row
 * fragment, which nests the record reference, and the item and page fragments
 * ride inside it. getFragmentData unwraps only the outermost mask in its type.
 */
export type CaptureBatchRow = UnmaskFragments<CaptureBatchRowFieldsFragment>;
export type CaptureBatchDetail = UnmaskFragments<CaptureBatchDetailFieldsFragment>;
export type CaptureItem = UnmaskFragments<CaptureItemFieldsFragment>;
export type CapturePage = UnmaskFragments<CapturePageFieldsFragment>;
export type CaptureRecordRef = UnmaskFragments<CaptureRecordRefFieldsFragment>;
export type CaptureDevice = UnmaskFragments<CaptureDeviceFieldsFragment>;
export type CaptureFleetDevice = UnmaskFragments<CaptureDeviceFleetFieldsFragment>;
export type CaptureSourceInfo = UnmaskFragments<CaptureSourceInfoFieldsFragment>;
export type CaptureProfile = UnmaskFragments<CaptureProfileFieldsFragment>;
export type CaptureRequest = UnmaskFragments<CaptureRequestFieldsFragment>;
export type CapturePairingPreview = CaptureDevicePairingQuery["captureDevicePairing"];
export type IssuedCoverSheet = CreateCaptureCoverSheetsMutation["createCaptureCoverSheets"][number];
export type {
  CaptureBatchSort,
  CaptureBatchStatus,
  CaptureCoverSheetInput,
  CaptureDeviceStatus,
  CaptureItemStatus,
  CapturePixelType,
  CaptureProfileInput,
  CaptureProfileStatus,
  CaptureRequestMode,
  CaptureRequestStatus,
  CaptureSeparatorStrategy,
  CaptureSource,
  CaptureSuggestionSource,
  CreateCaptureRequestInput,
  EditCaptureItemsInput,
  FileCaptureItemInput,
  FileCaptureItemsEntryInput,
};

type RequestOptions = { signal?: AbortSignal };

/*
 * getFragmentData hands back the object unmasked at runtime, but types it as
 * the masked fragment. The cast is the type catching up with the call.
 */
function unmasked<T>(value: unknown): T {
  return value as T;
}

function batchRow(masked: FragmentType<typeof CaptureBatchRowFieldsFragmentDoc>): CaptureBatchRow {
  return unmasked<CaptureBatchRow>(getFragmentData(CaptureBatchRowFieldsFragmentDoc, masked));
}

function batchDetail(
  masked: FragmentType<typeof CaptureBatchDetailFieldsFragmentDoc>,
): CaptureBatchDetail {
  return unmasked<CaptureBatchDetail>(
    getFragmentData(CaptureBatchDetailFieldsFragmentDoc, masked),
  );
}

function item(masked: FragmentType<typeof CaptureItemFieldsFragmentDoc>): CaptureItem {
  return unmasked<CaptureItem>(getFragmentData(CaptureItemFieldsFragmentDoc, masked));
}

function device(masked: FragmentType<typeof CaptureDeviceFieldsFragmentDoc>): CaptureDevice {
  return unmasked<CaptureDevice>(getFragmentData(CaptureDeviceFieldsFragmentDoc, masked));
}

function fleetDevice(
  masked: FragmentType<typeof CaptureDeviceFleetFieldsFragmentDoc>,
): CaptureFleetDevice {
  return unmasked<CaptureFleetDevice>(
    getFragmentData(CaptureDeviceFleetFieldsFragmentDoc, masked),
  );
}

function profile(masked: FragmentType<typeof CaptureProfileFieldsFragmentDoc>): CaptureProfile {
  return unmasked<CaptureProfile>(getFragmentData(CaptureProfileFieldsFragmentDoc, masked));
}

function request(masked: FragmentType<typeof CaptureRequestFieldsFragmentDoc>): CaptureRequest {
  return unmasked<CaptureRequest>(getFragmentData(CaptureRequestFieldsFragmentDoc, masked));
}

/** How many batches one request of the intake queue asks for. */
export const CAPTURE_BATCH_PAGE_SIZE = 30;

export type CaptureBatchFilter = {
  sort?: CaptureBatchSort;
  statuses?: CaptureBatchStatus[];
  source?: CaptureSource | null;
  mine?: boolean;
  targetType?: string | null;
  targetId?: string | null;
  query?: string | null;
  after?: string | null;
  first?: number;
};

export type CaptureBatchPage = {
  batches: CaptureBatchRow[];
  endCursor: string | null;
  hasNextPage: boolean;
  totalCount: number | null;
};

export async function fetchCaptureBatches(
  filter: CaptureBatchFilter = {},
  options?: RequestOptions,
): Promise<CaptureBatchPage> {
  const data = await requestGraphQL({
    document: CaptureBatchesDocument,
    operationName: "CaptureBatches",
    variables: {
      input: {
        first: filter.first ?? CAPTURE_BATCH_PAGE_SIZE,
        after: filter.after ?? null,
        sort: filter.sort ?? "Newest",
        statuses: filter.statuses ?? [],
        source: filter.source ?? null,
        mine: filter.mine ?? false,
        targetType: filter.targetType ?? null,
        targetId: filter.targetId ?? null,
        query: filter.query ?? null,
      },
    },
    signal: options?.signal,
  });

  return {
    batches: data.captureBatches.edges.map((edge) => batchRow(edge.node)),
    endCursor: data.captureBatches.pageInfo.endCursor ?? null,
    hasNextPage: data.captureBatches.pageInfo.hasNextPage,
    totalCount: data.captureBatches.totalCount ?? null,
  };
}

export async function fetchCaptureBatch(
  id: string,
  options?: RequestOptions,
): Promise<CaptureBatchDetail> {
  const data = await requestGraphQL({
    document: CaptureBatchDocument,
    operationName: "CaptureBatch",
    variables: { id },
    signal: options?.signal,
  });

  return batchDetail(data.captureBatch);
}

export async function fetchMyCaptureDevices(
  status: CaptureDeviceStatus | null,
  options?: RequestOptions,
): Promise<CaptureDevice[]> {
  const data = await requestGraphQL({
    document: MyCaptureDevicesDocument,
    operationName: "MyCaptureDevices",
    variables: { status },
    signal: options?.signal,
  });

  return data.myCaptureDevices.map(device);
}

export async function fetchCaptureDevices(
  filter: { status: CaptureDeviceStatus | null; query: string | null },
  options?: RequestOptions,
): Promise<CaptureFleetDevice[]> {
  const data = await requestGraphQL({
    document: CaptureDevicesDocument,
    operationName: "CaptureDevices",
    variables: filter,
    signal: options?.signal,
  });

  return data.captureDevices.map(fleetDevice);
}

export async function fetchAvailableCaptureProfiles(
  options?: RequestOptions,
): Promise<CaptureProfile[]> {
  const data = await requestGraphQL({
    document: AvailableCaptureProfilesDocument,
    operationName: "AvailableCaptureProfiles",
    variables: {},
    signal: options?.signal,
  });

  return data.availableCaptureProfiles.map(profile);
}

export async function fetchCaptureProfiles(
  filter: { status: CaptureProfileStatus | null; query: string | null },
  options?: RequestOptions,
): Promise<CaptureProfile[]> {
  const data = await requestGraphQL({
    document: CaptureProfilesDocument,
    operationName: "CaptureProfiles",
    variables: filter,
    signal: options?.signal,
  });

  return data.captureProfiles.map(profile);
}

/** How many recent requests into one record its Documents tab reads. */
export const CAPTURE_TARGET_REQUEST_LIMIT = 10;

export async function fetchCaptureRequestsForTarget(
  target: { targetType: string; targetId: string },
  options?: RequestOptions,
): Promise<CaptureRequest[]> {
  const data = await requestGraphQL({
    document: CaptureRequestsForTargetDocument,
    operationName: "CaptureRequestsForTarget",
    variables: { ...target, limit: CAPTURE_TARGET_REQUEST_LIMIT },
    signal: options?.signal,
  });

  return data.captureRequestsForTarget.map(request);
}

export async function fetchCapturePairing(
  userCode: string,
  options?: RequestOptions,
): Promise<CapturePairingPreview> {
  const data = await requestGraphQL({
    document: CaptureDevicePairingDocument,
    operationName: "CaptureDevicePairing",
    variables: { userCode },
    signal: options?.signal,
  });

  return data.captureDevicePairing;
}

export async function editCaptureItems(
  batchId: string,
  input: EditCaptureItemsInput,
): Promise<CaptureBatchDetail> {
  const data = await requestGraphQL({
    document: EditCaptureItemsDocument,
    operationName: "EditCaptureItems",
    variables: { batchId, input },
  });

  return batchDetail(data.editCaptureItems);
}

export async function fileCaptureItem(
  id: string,
  input: FileCaptureItemInput,
): Promise<CaptureItem> {
  const data = await requestGraphQL({
    document: FileCaptureItemDocument,
    operationName: "FileCaptureItem",
    variables: { id, input },
  });

  return item(data.fileCaptureItem);
}

export type FileCaptureItemsResult = {
  filed: CaptureItem[];
  failures: { itemId: string; message: string }[];
};

export async function fileCaptureItems(
  items: FileCaptureItemsEntryInput[],
): Promise<FileCaptureItemsResult> {
  const data = await requestGraphQL({
    document: FileCaptureItemsDocument,
    operationName: "FileCaptureItems",
    variables: { items },
  });

  return {
    filed: data.fileCaptureItems.filed.map(item),
    failures: data.fileCaptureItems.failures,
  };
}

export async function discardCaptureItem(id: string, version: number): Promise<CaptureBatchRow> {
  const data = await requestGraphQL({
    document: DiscardCaptureItemDocument,
    operationName: "DiscardCaptureItem",
    variables: { id, version },
  });

  return batchRow(data.discardCaptureItem);
}

export async function discardCaptureBatch(id: string, version: number): Promise<CaptureBatchRow> {
  const data = await requestGraphQL({
    document: DiscardCaptureBatchDocument,
    operationName: "DiscardCaptureBatch",
    variables: { id, version },
  });

  return batchRow(data.discardCaptureBatch);
}

export async function createCaptureRequest(
  input: CreateCaptureRequestInput,
): Promise<CaptureRequest> {
  const data = await requestGraphQL({
    document: CreateCaptureRequestDocument,
    operationName: "CreateCaptureRequest",
    variables: { input },
  });

  return request(data.createCaptureRequest);
}

export async function cancelCaptureRequest(id: string): Promise<CaptureRequest> {
  const data = await requestGraphQL({
    document: CancelCaptureRequestDocument,
    operationName: "CancelCaptureRequest",
    variables: { id },
  });

  return request(data.cancelCaptureRequest);
}

export async function createCaptureCoverSheets(
  sheets: CaptureCoverSheetInput[],
): Promise<IssuedCoverSheet[]> {
  const data = await requestGraphQL({
    document: CreateCaptureCoverSheetsDocument,
    operationName: "CreateCaptureCoverSheets",
    variables: { sheets },
  });

  return data.createCaptureCoverSheets;
}

export async function approveCapturePairing(
  userCode: string,
  deviceName: string | null,
): Promise<void> {
  await requestGraphQL({
    document: ApproveCaptureDevicePairingDocument,
    operationName: "ApproveCaptureDevicePairing",
    variables: { userCode, deviceName },
  });
}

export async function denyCapturePairing(userCode: string): Promise<void> {
  await requestGraphQL({
    document: DenyCaptureDevicePairingDocument,
    operationName: "DenyCaptureDevicePairing",
    variables: { userCode },
  });
}

export async function revokeMyCaptureDevice(
  id: string,
  reason: string | null,
): Promise<CaptureDevice> {
  const data = await requestGraphQL({
    document: RevokeMyCaptureDeviceDocument,
    operationName: "RevokeMyCaptureDevice",
    variables: { id, reason },
  });

  return device(data.revokeMyCaptureDevice);
}

export async function revokeCaptureDevice(
  id: string,
  reason: string | null,
): Promise<CaptureFleetDevice> {
  const data = await requestGraphQL({
    document: RevokeCaptureDeviceDocument,
    operationName: "RevokeCaptureDevice",
    variables: { id, reason },
  });

  return fleetDevice(data.revokeCaptureDevice);
}

export async function createCaptureProfile(input: CaptureProfileInput): Promise<CaptureProfile> {
  const data = await requestGraphQL({
    document: CreateCaptureProfileDocument,
    operationName: "CreateCaptureProfile",
    variables: { input },
  });

  return profile(data.createCaptureProfile);
}

export async function updateCaptureProfile(
  id: string,
  version: number,
  input: CaptureProfileInput,
): Promise<CaptureProfile> {
  const data = await requestGraphQL({
    document: UpdateCaptureProfileDocument,
    operationName: "UpdateCaptureProfile",
    variables: { id, version, input },
  });

  return profile(data.updateCaptureProfile);
}

export async function deleteCaptureProfile(id: string): Promise<void> {
  await requestGraphQL({
    document: DeleteCaptureProfileDocument,
    operationName: "DeleteCaptureProfile",
    variables: { id },
  });
}
