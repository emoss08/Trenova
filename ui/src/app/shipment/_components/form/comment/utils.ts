import { CommentType } from "@/lib/schemas/shipment-comment-schema";

export const COMMENT_TYPES = [
  {
    value: CommentType.enum.hot,
    label: "Hot",
    icon: "🔥",
    description: "Urgent or critical",
    className:
      "bg-red-400/30 text-red-600 border border-red-600/30 dark:bg-red-600/30 dark:text-red-400 dark:border-red-400/30",
    textAreaClassName:
      "border-red-600/30 bg-red-600/10 px-2.5 py-1.5 min-h-[60px] max-h-[200px] focus-visible:border-red-500 focus-visible:ring-2 focus-visible:ring-red-500/20",
  },
  {
    value: CommentType.enum.billing,
    label: "Billing",
    icon: "💰",
    description: "Billing related",
    className:
      "bg-green-400/30 text-green-600 border border-green-600/30 dark:bg-green-600/30 dark:text-green-400 dark:border-green-400/30",
    textAreaClassName:
      "border-green-600/30 bg-green-600/10 px-2.5 py-1.5 min-h-[60px] max-h-[200px] focus-visible:border-green-500 focus-visible:ring-2 focus-visible:ring-green-500/20",
  },
  {
    value: CommentType.enum.dispatch,
    label: "Dispatch",
    icon: "🚚",
    description: "Dispatch related",
    className:
      "bg-blue-400/30 text-blue-600 border border-blue-600/30 dark:bg-blue-600/30 dark:text-blue-400 dark:border-blue-400/30",
    textAreaClassName:
      "border-blue-600/30 bg-blue-600/10 px-2.5 py-1.5 min-h-[60px] max-h-[200px] focus-visible:border-blue-500 focus-visible:ring-2 focus-visible:ring-blue-500/20",
  },
] as const;

export type CommentType = (typeof COMMENT_TYPES)[number]["value"];

export const getCommentTypeClassName = (commentType?: CommentType | null) => {
  if (!commentType) {
    return "";
  }

  const commentTypeData = COMMENT_TYPES.find(
    (type) => type.value === commentType,
  );
  return commentTypeData?.textAreaClassName;
};
