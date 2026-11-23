import { appPath } from "@/lib/app-path";

/** The notification kinds the server sends a requester about their export of Desk conversations. */
export const DESK_TRANSCRIPT_EXPORT_READY_EVENT = "desk_transcript_export_ready";
export const DESK_TRANSCRIPT_EXPORT_FAILED_EVENT = "desk_transcript_export_failed";

/** Where the list of everyone's Desk conversations opens, and its exports. */
export const DESK_CONVERSATIONS_PATH = appPath("/admin/desk-conversations");
export const DESK_TRANSCRIPT_EXPORTS_PATH = `${DESK_CONVERSATIONS_PATH}?view=exports`;

/** The most conversations a person may pick from the list and take out at once. */
export const MAX_SELECTED_CONVERSATIONS = 200;
