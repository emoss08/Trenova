/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { ShipmentCommentSchema } from "@/lib/schemas/shipment-comment-schema";
import { create } from "zustand";

interface CommentEditState {
  editingComment: ShipmentCommentSchema | null;
  isEditMode: boolean;
  setEditingComment: (comment: ShipmentCommentSchema | null) => void;
  clearEditMode: () => void;
}

export const useCommentEditStore = create<CommentEditState>((set) => ({
  editingComment: null,
  isEditMode: false,
  setEditingComment: (comment) =>
    set({
      editingComment: comment,
      isEditMode: !!comment,
    }),
  clearEditMode: () =>
    set({
      editingComment: null,
      isEditMode: false,
    }),
}));