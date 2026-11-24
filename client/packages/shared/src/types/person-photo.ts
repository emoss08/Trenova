import { z } from "zod";

/**
 * A person's photo (a worker's or a user's) as the server signs it: the picture and a thumbnail
 * of it. The links expire, so they are read again with the record rather than kept.
 */
export const personPhotoSchema = z.object({
  url: z.string(),
  thumbnailUrl: z.string(),
});

export type PersonPhoto = z.infer<typeof personPhotoSchema>;

/**
 * What replacing or removing a worker's photo answers: the new links (null once removed) and the
 * worker's version after the change, which the next save of the worker must carry.
 */
export const workerPhotoChangeSchema = z.object({
  url: z.string().nullable(),
  thumbnailUrl: z.string().nullable(),
  version: z.number().int(),
});

export type WorkerPhotoChange = z.infer<typeof workerPhotoChangeSchema>;

/** The photo a change leaves the worker with, or null when it left them without one. */
export function photoAfterChange(change: WorkerPhotoChange): PersonPhoto | null {
  return change.url && change.thumbnailUrl
    ? { url: change.url, thumbnailUrl: change.thumbnailUrl }
    : null;
}

/** A photo a driver sent from the portal, waiting for the office to decide on it. */
export const portalPhotoRequestSchema = z.object({
  id: z.string(),
  status: z.string(),
  submittedAt: z.number().int(),
  proposedPhoto: personPhotoSchema.nullable(),
});

export type PortalPhotoRequest = z.infer<typeof portalPhotoRequestSchema>;
