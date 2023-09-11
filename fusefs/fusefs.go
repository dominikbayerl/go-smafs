package fusefs

import (
	"context"
	"hash/fnv"
	"path"
	"path/filepath"
	"syscall"

	"github.com/dominikbayerl/go-smafs/sma"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type FuseRoot struct {
	ctx     context.Context
	api     *sma.SMAApi
	counter uint
}

type FuseNode struct {
	fs.Inode
	root    *FuseRoot
	Size    uint64
	content []byte
}

func NewFuseFS(ctx context.Context, api *sma.SMAApi) *FuseNode {
	return &FuseNode{root: &FuseRoot{ctx: ctx, api: api, counter: 0}}

}

var _ = (fs.NodeGetattrer)((*FuseNode)(nil))

func (r *FuseNode) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = 0755
	return 0
}

var _ = (fs.NodeReaddirer)((*FuseNode)(nil))
var _ = (fs.NodeLookuper)((*FuseNode)(nil))

func (r *FuseRoot) ReserveIno() uint64 {
	r.counter++
	return uint64(r.counter)
}

func (r *FuseRoot) MakeIno(name string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(name))
	return h.Sum64()
}

func (r *FuseNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	parentDir := path.Join("/", r.Path(nil))

	entries, err := r.root.api.GetFS(r.root.ctx, parentDir)
	if err != nil {
		return nil, syscall.EFAULT
	}
	v := make([]fuse.DirEntry, len(entries))
	for idx, entry := range entries {
		if entry.Filename != "" {
			v[idx] = fuse.DirEntry{Mode: fuse.S_IFREG, Name: entry.Filename, Ino: r.root.MakeIno(entry.Filename)}
		} else if entry.DirectoryName != "" {
			v[idx] = fuse.DirEntry{Mode: fuse.S_IFDIR, Name: entry.DirectoryName, Ino: r.root.MakeIno(entry.DirectoryName)}
		}
	}
	return fs.NewListDirStream(v), 0
}

func (r *FuseNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	mode := syscall.S_IFDIR
	// TODO: This is a heuristic and should be fixed.
	// The mode should be passed somehow from the Readdir()-enumeration
	if filepath.Ext(name) != "" {
		mode = syscall.S_IFREG
	}
	out.Mode = 0755
	out.Size = 0
	return r.NewInode(ctx, &FuseNode{root: r.root}, fs.StableAttr{Mode: uint32(mode), Ino: r.root.MakeIno(name)}), 0
}

type bytesFileHandle struct {
	content []byte
}

// bytesFileHandle allows reads
var _ = (fs.FileReader)((*bytesFileHandle)(nil))

func (fh *bytesFileHandle) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	end := off + int64(len(dest))
	if end > int64(len(fh.content)) {
		end = int64(len(fh.content))
	}

	// We could copy to the `dest` buffer, but since we have a
	// []byte already, return that.
	return fuse.ReadResultData(fh.content[off:end]), 0
}

// Implement (handleless) Open
var _ = (fs.NodeOpener)((*FuseNode)(nil))

func (r *FuseNode) Open(ctx context.Context, openFlags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	// disallow writes
	if fuseFlags&(syscall.O_RDWR|syscall.O_WRONLY) != 0 {
		return nil, 0, syscall.EROFS
	}

	content, err := r.root.api.Download(r.root.ctx, r.Path(nil))
	if err != nil {
		return nil, 0, syscall.EFAULT
	}
	fh = &bytesFileHandle{
		content: content,
	}

	// Return FOPEN_DIRECT_IO so content is not cached.
	return fh, fuse.FOPEN_DIRECT_IO, 0
}
