import { MOVE_DELETE_DIALOG_KEY } from "@/constants/env";

// Utility function to resequence moves after deletion
const resequenceMoves = <T extends { sequence: number }>(
  moves: T[],
  deletedIndex: number,
): T[] => {
  // Create a copy of the moves array to avoid mutating the original
  const updatedMoves = [...moves];

  // Get the sequence number of the deleted move
  const deletedSequence = moves[deletedIndex].sequence;

  // Adjust sequence numbers for all moves after the deleted one
  for (let i = 0; i < updatedMoves.length; i++) {
    if (i !== deletedIndex && updatedMoves[i].sequence > deletedSequence) {
      updatedMoves[i] = {
        ...updatedMoves[i],
        sequence: updatedMoves[i].sequence - 1,
      };
    }
  }

  return updatedMoves;
};

