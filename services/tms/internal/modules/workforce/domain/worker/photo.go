package worker

func (w *Worker) HasPhoto() bool {
	return w.PhotoKey != ""
}

func (w *Worker) SetPhoto(key, thumbnailKey string) {
	w.PhotoKey = key
	w.PhotoThumbnailKey = thumbnailKey
}

func (w *Worker) KeepPhoto(stored *Worker) {
	if stored == nil {
		w.SetPhoto("", "")
		return
	}
	w.SetPhoto(stored.PhotoKey, stored.PhotoThumbnailKey)
}
