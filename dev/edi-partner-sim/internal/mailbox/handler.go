package mailbox

import (
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"

	"github.com/pkg/sftp"
)

type handler struct {
	box     *Box
	watcher Watcher
	logger  *slog.Logger
}

func (h *handler) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	return h.box.root.Open(relative(r.Filepath))
}

func (h *handler) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	flags := os.O_WRONLY | os.O_CREATE
	pflags := r.Pflags()
	if pflags.Trunc {
		flags |= os.O_TRUNC
	}
	if pflags.Excl {
		flags |= os.O_EXCL
	}
	file, err := h.box.root.OpenFile(relative(r.Filepath), flags, 0o644)
	if err != nil {
		return nil, err
	}
	return &watchedFile{File: file, handler: h, path: path.Clean(r.Filepath)}, nil
}

func (h *handler) Filecmd(r *sftp.Request) error {
	switch r.Method {
	case "Setstat":
		return nil
	case "Rename":
		return h.rename(r.Filepath, r.Target)
	case "Rmdir", "Remove":
		if err := h.box.root.Remove(relative(r.Filepath)); err != nil {
			return err
		}
		if r.Method == "Remove" {
			h.left(path.Clean(r.Filepath))
		}
		return nil
	case "Mkdir":
		return h.box.root.Mkdir(relative(r.Filepath), 0o755)
	default:
		return sftp.ErrSSHFxOpUnsupported
	}
}

func (h *handler) PosixRename(r *sftp.Request) error {
	return h.rename(r.Filepath, r.Target)
}

func (h *handler) rename(from, to string) error {
	if err := h.box.root.Rename(relative(from), relative(to)); err != nil {
		return err
	}
	from, to = path.Clean(from), path.Clean(to)
	if path.Dir(from) != path.Dir(to) {
		h.left(from)
		h.arrived(to)
	}
	return nil
}

func (h *handler) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	rel := relative(r.Filepath)
	switch r.Method {
	case "List":
		entries, err := fs.ReadDir(h.box.root.FS(), rel)
		if err != nil {
			return nil, err
		}
		infos := make([]os.FileInfo, 0, len(entries))
		for _, entry := range entries {
			info, infoErr := entry.Info()
			if infoErr != nil {
				continue
			}
			infos = append(infos, info)
		}
		return listerAt(infos), nil
	case "Stat":
		info, err := h.box.root.Stat(rel)
		if err != nil {
			return nil, err
		}
		return listerAt{info}, nil
	default:
		return nil, sftp.ErrSSHFxOpUnsupported
	}
}

func (h *handler) arrived(p string) {
	if h.watcher == nil || path.Dir(p) != h.box.Outbound {
		return
	}
	contents, err := h.box.root.ReadFile(relative(p))
	if err != nil {
		h.logger.Warn("a delivered file could not be read", "path", p, "error", err)
		return
	}
	h.watcher.Delivered(h.box.User, path.Base(p), contents)
}

func (h *handler) left(p string) {
	if h.watcher != nil && path.Dir(p) == h.box.Inbound {
		h.watcher.PickedUp(h.box.User, path.Base(p))
	}
}

type watchedFile struct {
	*os.File
	handler *handler
	path    string
}

func (f *watchedFile) Close() error {
	if err := f.File.Close(); err != nil {
		return err
	}
	f.handler.arrived(f.path)
	return nil
}

type listerAt []os.FileInfo

func (l listerAt) ListAt(out []os.FileInfo, offset int64) (int, error) {
	if offset >= int64(len(l)) {
		return 0, io.EOF
	}
	n := copy(out, l[offset:])
	if n < len(out) {
		return n, io.EOF
	}
	return n, nil
}
