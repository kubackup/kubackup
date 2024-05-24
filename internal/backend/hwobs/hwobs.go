package hwobs

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/errors"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"hash"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

var _ restic.Backend = &HwObs{}

const defaultLayout = "default"

type HwObs struct {
	client *obs.ObsClient
	sem    *backend.Semaphore
	cfg    Config
	backend.Layout
}

type fileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (fi *fileInfo) Name() string       { return fi.name }    // base name of the file
func (fi *fileInfo) Size() int64        { return fi.size }    // length in bytes for regular files; system-dependent for others
func (fi *fileInfo) Mode() os.FileMode  { return fi.mode }    // file mode bits
func (fi *fileInfo) ModTime() time.Time { return fi.modTime } // modification time
func (fi *fileInfo) IsDir() bool        { return fi.isDir }   // abbreviation for Mode().IsDir()
func (fi *fileInfo) Sys() interface{}   { return nil }        // underlying data source (can return nil)

func (h *HwObs) ReadDir(ctx context.Context, dir string) (list []os.FileInfo, err error) {
	if dir[len(dir)-1] != '/' {
		dir += "/"
	}

	input := &obs.ListObjectsInput{
		Bucket: h.cfg.BucketName,
	}
	input.Prefix = dir
	output, err := h.client.ListObjects(input)
	if h.IsAccessDenied(err) {
		return nil, fmt.Errorf("权限不足")
	}
	if h.IsNotExist(err) {
		return nil, fmt.Errorf("404")
	}
	if err != nil {
		return nil, err
	}
	contents := output.Contents
	for _, con := range contents {
		if con.Key == "" {
			continue
		}
		name := strings.TrimPrefix(con.Key, dir)
		if name == "" {
			continue
		}
		entry := &fileInfo{
			name:    name,
			size:    con.Size,
			modTime: con.LastModified,
		}
		if name[len(name)-1] == '/' {
			entry.isDir = true
			entry.mode = os.ModeDir | 0755
			entry.name = name[:len(name)-1]
		} else {
			entry.mode = 0644
		}

		list = append(list, entry)
	}
	return list, nil
}

func Open(ctx context.Context, cfg Config, rt http.RoundTripper) (restic.Backend, error) {
	return open(ctx, cfg, rt)
}

func Create(ctx context.Context, cfg Config, rt http.RoundTripper) (restic.Backend, error) {
	be, err := open(ctx, cfg, rt)
	if err != nil {
		return nil, errors.Wrap(err, "open")
	}
	_, err = be.client.HeadBucket(cfg.BucketName)
	if be.IsNotExist(err) {
		return nil, errors.Wrap(err, "Bucket不存在，请在华为OBS控制台新建桶")
	}
	return be, nil
}
func open(ctx context.Context, cfg Config, rt http.RoundTripper) (*HwObs, error) {
	configssl := obs.WithSslVerify(cfg.SslEnable)
	timeout := obs.WithConnectTimeout(5)
	client, err := obs.New(cfg.Ak, cfg.Sk, cfg.Endpoint, configssl, timeout)
	if err != nil {
		return nil, errors.Wrap(err, "obs.New")
	}
	sem, err := backend.NewSemaphore(cfg.Connections)
	if err != nil {
		return nil, err
	}
	be := &HwObs{
		client: client,
		sem:    sem,
		cfg:    cfg,
	}

	l, err := backend.ParseLayout(ctx, be, cfg.Layout, defaultLayout, cfg.Prefix)
	if err != nil {
		return nil, err
	}

	be.Layout = l

	return be, nil
}

func (h *HwObs) Location() string {
	return h.Join(h.cfg.BucketName, h.cfg.Prefix)
}

func (h *HwObs) Hasher() hash.Hash {
	return nil
}

func (h *HwObs) Test(ctx context.Context, handle restic.Handle) (bool, error) {
	objName := h.Filename(handle)
	h.sem.GetToken()
	defer h.sem.ReleaseToken()
	input := &obs.HeadObjectInput{
		Bucket: h.cfg.BucketName,
		Key:    objName,
	}
	_, err := h.client.HeadObject(input)
	if h.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, err
}

func (h *HwObs) Remove(ctx context.Context, handle restic.Handle) error {
	objName := h.Filename(handle)
	h.sem.GetToken()
	defer h.sem.ReleaseToken()
	input := &obs.DeleteObjectInput{
		Bucket: h.cfg.BucketName,
		Key:    objName,
	}
	_, err := h.client.DeleteObject(input)
	if h.IsAccessDenied(err) {
		return fmt.Errorf("权限不足")
	}
	if h.IsNotExist(err) {
		return fmt.Errorf("404")
	}
	if err != nil {
		return err
	}
	return errors.Wrap(err, "client.DeleteObject")
}

func (h *HwObs) Close() error {
	return nil
}

func (h *HwObs) Save(ctx context.Context, handle restic.Handle, rd restic.RewindReader) error {
	if err := handle.Valid(); err != nil {
		return backoff.Permanent(err)
	}

	objName := h.Filename(handle)

	h.sem.GetToken()
	defer h.sem.ReleaseToken()

	input := &obs.PutObjectInput{}
	input.ContentType = "application/octet-stream"
	input.Key = objName
	input.Bucket = h.cfg.BucketName
	input.StorageClass = obs.ParseStringToStorageClassType(h.cfg.StorageClass)
	input.ContentLength = rd.Length()
	input.ContentMD5 = base64.StdEncoding.EncodeToString(rd.Hash())
	input.Body = ioutil.NopCloser(rd)

	_, err := h.client.PutObject(input)
	if h.IsAccessDenied(err) {
		return fmt.Errorf("权限不足")
	}
	if err != nil {
		return err
	}
	return errors.Wrap(err, "client.PutObject")
}

func (h *HwObs) Load(ctx context.Context, handle restic.Handle, length int, offset int64, fn func(rd io.Reader) error) error {
	return backend.DefaultLoad(ctx, handle, length, offset, h.openReader, fn)
}

func (h *HwObs) openReader(ctx context.Context, handle restic.Handle, length int, offset int64) (io.ReadCloser, error) {

	if err := handle.Valid(); err != nil {
		return nil, backoff.Permanent(err)
	}

	if offset < 0 {
		return nil, errors.New("offset is negative")
	}

	if length < 0 {
		return nil, errors.Errorf("invalid length %d", length)
	}

	objName := h.Filename(handle)
	input := &obs.GetObjectInput{}
	input.Bucket = h.cfg.BucketName
	input.Key = objName
	var err error
	if length > 0 {
		input.RangeStart = offset
		input.RangeEnd = offset + int64(length) - 1
	} else if offset > 0 {
		input.RangeStart = offset
		// 华为OBS 如果指定的范围无效（比如开始位置、结束位置为负数，大于文件大小），则会返回整个对象。
		// RangeEnd 指定下载对象的结束位置。如果该值大于对象长度-1，实际仍取对象长度-1。
		// 当length大于文件长度时，length传入值为0
		input.RangeEnd = math.MaxInt64
	}
	h.sem.GetToken()
	object, err := h.client.GetObject(input)
	if err != nil {
		h.sem.ReleaseToken()
		return nil, err
	}
	rd := object.Body

	closeRd := wrapReader{
		ReadCloser: rd,
		f: func() {
			h.sem.ReleaseToken()
		},
	}

	return closeRd, err
}

type wrapReader struct {
	io.ReadCloser
	f func()
}

func (wr wrapReader) Close() error {
	err := wr.ReadCloser.Close()
	wr.f()
	return err
}

func (h *HwObs) Stat(ctx context.Context, handle restic.Handle) (restic.FileInfo, error) {
	objName := h.Filename(handle)
	input := &obs.GetObjectMetadataInput{
		Bucket: h.cfg.BucketName,
		Key:    objName,
	}
	h.sem.GetToken()
	defer h.sem.ReleaseToken()
	object, err := h.client.GetObjectMetadata(input)
	if err != nil {
		return restic.FileInfo{}, err
	}
	return restic.FileInfo{Size: object.ContentLength, Name: handle.Name}, nil
}

func (h *HwObs) List(ctx context.Context, t restic.FileType, fn func(restic.FileInfo) error) error {
	prefix, _ := h.Basedir(t)

	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	input := &obs.ListObjectsInput{
		Bucket: h.cfg.BucketName,
	}
	for {
		input.Prefix = prefix
		output, err := h.client.ListObjects(input)
		if h.IsAccessDenied(err) {
			return fmt.Errorf("权限不足")
		}
		if h.IsNotExist(err) {
			return fmt.Errorf("404")
		}
		if err != nil {
			return err
		}
		contents := output.Contents
		marker := ""
		for i, con := range contents {
			// obs 默认最大返回1000条数据，通过设置input.Marker来获取后面数据
			// Marker 列举对象的起始位置，返回的对象列表将是对象名按照字典序排序后该参数以后的所有对象。
			if i == 999 {
				marker = con.Key
			}
			m := strings.TrimPrefix(con.Key, prefix)
			if m == "" {
				continue
			}
			fi := restic.FileInfo{
				Name: path.Base(m),
				Size: con.Size,
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			err := fn(fi)
			if err != nil {
				return err
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
		}
		if marker == "" {
			return ctx.Err()
		} else {
			input.Marker = marker
		}

	}
}

func (h *HwObs) IsNotExist(err error) bool {
	if err == nil {
		return false
	}
	if obsError, ok := err.(obs.ObsError); ok {
		if obsError.StatusCode == 404 {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

// IsAccessDenied returns true if the error is caused by Access Denied.
func (h *HwObs) IsAccessDenied(err error) bool {
	if err == nil {
		return false
	}
	if obsError, ok := err.(obs.ObsError); ok {
		if obsError.StatusCode == 403 {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

// Remove keys for a specified backend type.
func (h *HwObs) removeKeys(ctx context.Context, t restic.FileType) error {
	return h.List(ctx, restic.PackFile, func(fi restic.FileInfo) error {
		return h.Remove(ctx, restic.Handle{Type: t, Name: fi.Name})
	})
}
func (h *HwObs) Delete(ctx context.Context) error {
	alltypes := []restic.FileType{
		restic.PackFile,
		restic.KeyFile,
		restic.LockFile,
		restic.SnapshotFile,
		restic.IndexFile}

	for _, t := range alltypes {
		err := h.removeKeys(ctx, t)
		if err != nil {
			return nil
		}
	}

	return h.Remove(ctx, restic.Handle{Type: restic.ConfigFile})
}

// Join combines path components with slashes.
func (be *HwObs) Join(p ...string) string {
	return path.Join(p...)
}
