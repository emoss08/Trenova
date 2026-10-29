package chaptertitles

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
)

func SnapToChapter(
	older []conversation.Message,
	through int,
	titles map[int]string,
) (snapped []conversation.Message, snappedThrough int) {
	if len(titles) == 0 || len(older) == 0 {
		return older, through
	}

	first := -1
	spoken := 0
	for idx := range older {
		if older[idx].Compacted() {
			continue
		}
		if first < 0 {
			first = older[idx].Sequence
		}
		spoken++
	}
	if first < 0 {
		return older, through
	}

	start := -1
	for sequence := range titles {
		if sequence > first && sequence <= through && sequence > start {
			start = sequence
		}
	}
	if start < 0 {
		return older, through
	}

	cut := len(older)
	for idx := range older {
		if !older[idx].Compacted() && older[idx].Sequence >= start {
			cut = idx

			break
		}
	}

	kept := 0
	for idx := range older[:cut] {
		if !older[idx].Compacted() {
			kept++
		}
	}
	if kept == 0 || kept*2 < spoken {
		return older, through
	}

	return older[:cut], older[cut-1].Sequence
}
