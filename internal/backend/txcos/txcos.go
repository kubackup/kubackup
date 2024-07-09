package txcos

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend/layout"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend/location"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/errors"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"github.com/tencentyun/cos-go-sdk-v5"
	"hash"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

var _ restic.Backend = &TxCos{}

const defaultLayout = "default"

type TxCos struct {
	client *cos.Client
	cfg    Config
	layout.Layout
}

func (t *TxCos) Connections() uint {
	return t.cfg.Connections
}

func (t *TxCos) HasAtomicReplace() bool {
	return true
}

func NewFactory() location.Factory {
	return location.NewHTTPBackendFactory("cos", ParseConfig, location.NoPassword, Create, Open)
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

func (t *TxCos) ReadDir(ctx context.Context, dir string) (list []os.FileInfo, err error) {
	if dir[len(dir)-1] != '/' {
		dir += "/"
	}
	var marker string
	opt := &cos.BucketGetOptions{
		Prefix:    dir,
		Delimiter: "/",
		MaxKeys:   1000,
	}

	isTruncated := true
	for isTruncated {
		opt.Marker = marker
		v, _, err := t.client.Bucket.Get(ctx, opt)
		if t.IsAccessDenied(err) {
			return nil, fmt.Errorf("权限不足")
		}
		if t.IsNotExist(err) {
			return nil, fmt.Errorf("资源不存在")
		}
		if err != nil {
			return nil, err
		}
		for _, con := range v.Contents {
			if con.Key == "" {
				continue
			}
			name := strings.TrimPrefix(con.Key, dir)
			if name == "" {
				continue
			}
			LastModified, err := time.Parse(time.RFC3339, con.LastModified)
			if err != nil {
				return nil, err
			}
			entry := &fileInfo{
				name:    name,
				size:    con.Size,
				modTime: LastModified,
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
		isTruncated = v.IsTruncated // 是否还有数据
		marker = v.NextMarker       // 设置下次请求的起始 key
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
	ok, err := be.client.Bucket.IsExist(ctx)
	if err == nil && ok {

	} else if err != nil {
		return nil, errors.Wrap(err, "head bucket failed")
	} else {
		return nil, errors.Wrap(err, "Bucket不存在，请在腾讯对象存储库控制台新建桶")
	}

	return be, nil
}
func open(ctx context.Context, cfg Config, rt http.RoundTripper) (*TxCos, error) {
	u, _ := url.Parse(cfg.Endpoint)
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			Transport: rt,
			SecretID:  cfg.SecretID,  // 用户的 SecretId，建议使用子账号密钥，授权遵循最小权限指引，降低使用风险。子账号密钥获取可参考 https://cloud.tencent.com/document/product/598/37140
			SecretKey: cfg.SecretKey, // 用户的 SecretKey，建议使用子账号密钥，授权遵循最小权限指引，降低使用风险。子账号密钥获取可参考 https://cloud.tencent.com/document/product/598/37140
		},
		Timeout: 5 * time.Second,
	})
	client.Conf.EnableCRC = cfg.EnableCRC
	be := &TxCos{
		client: client,
		cfg:    cfg,
	}

	l, err := layout.ParseLayout(ctx, be, cfg.Layout, defaultLayout, cfg.Prefix)
	if err != nil {
		return nil, err
	}

	be.Layout = l

	return be, nil
}

func (t *TxCos) Location() string {
	return t.Join(t.cfg.Endpoint, t.cfg.Prefix)
}

func (t *TxCos) Hasher() hash.Hash {
	return nil
}

func (t *TxCos) Remove(ctx context.Context, handle restic.Handle) error {
	objName := t.Filename(handle)
	_, err := t.client.Object.Delete(ctx, objName)
	if t.IsAccessDenied(err) {
		return fmt.Errorf("权限不足")
	}
	if t.IsNotExist(err) {
		return fmt.Errorf("资源不存在")
	}
	if err != nil {
		return err
	}
	return errors.Wrap(err, "client.DeleteObject")
}

func (t *TxCos) Close() error {
	return nil
}

func (t *TxCos) Save(ctx context.Context, handle restic.Handle, rd restic.RewindReader) error {
	if err := handle.Valid(); err != nil {
		return backoff.Permanent(err)
	}

	objName := t.Filename(handle)

	opt := &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			ContentType:      "application/octet-stream",
			ContentMD5:       base64.StdEncoding.EncodeToString(rd.Hash()),
			ContentLength:    rd.Length(),
			XCosStorageClass: t.cfg.StorageClass,
		},
	}

	_, err := t.client.Object.Put(ctx, objName, io.NopCloser(rd), opt)
	if t.IsAccessDenied(err) {
		return fmt.Errorf("权限不足")
	}
	if err != nil {
		return err
	}
	return errors.Wrap(err, "client.PutObject")
}

func (t *TxCos) Load(ctx context.Context, handle restic.Handle, length int, offset int64, fn func(rd io.Reader) error) error {
	return backend.DefaultLoad(ctx, handle, length, offset, t.openReader, fn)
}

func (t *TxCos) openReader(ctx context.Context, handle restic.Handle, length int, offset int64) (io.ReadCloser, error) {

	if err := handle.Valid(); err != nil {
		return nil, backoff.Permanent(err)
	}

	if offset < 0 {
		return nil, errors.New("offset is negative")
	}

	if length < 0 {
		return nil, errors.Errorf("invalid length %d", length)
	}

	objName := t.Filename(handle)

	var err error
	byteRange := fmt.Sprintf("bytes=%d-", offset)
	if length > 0 {
		byteRange = fmt.Sprintf("bytes=%d-%d", offset, offset+int64(length)-1)
	}
	opt := &cos.ObjectGetOptions{
		ResponseContentType: "application/octet-stream",
		Range:               byteRange, // 通过 range 下载0~3字节的数据
	}

	object, err := t.client.Object.Get(ctx, objName, opt)
	if err != nil {
		return nil, err
	}
	rd := object.Body

	return rd, err
}

func (t *TxCos) Stat(ctx context.Context, handle restic.Handle) (restic.FileInfo, error) {
	objName := t.Filename(handle)

	resp, err := t.client.Object.Head(ctx, objName, nil)
	if err != nil {
		return restic.FileInfo{}, err
	}
	length := resp.Header.Get("Content-Length")
	size, err := strconv.ParseInt(length, 10, 64)
	if err != nil {
		return restic.FileInfo{}, err
	}
	return restic.FileInfo{Size: size, Name: handle.Name}, nil
}

func (t *TxCos) listv1(ctx context.Context, prefix string, fn func(restic.FileInfo) error) error {

	var marker string
	opt := &cos.BucketGetOptions{
		Prefix:    prefix,
		Delimiter: "/",
		MaxKeys:   1000,
	}

	isTruncated := true
	for isTruncated {
		opt.Marker = marker
		v, _, err := t.client.Bucket.Get(ctx, opt)
		if t.IsAccessDenied(err) {
			return fmt.Errorf("权限不足")
		}
		if t.IsNotExist(err) {
			return fmt.Errorf("资源不存在")
		}
		if err != nil {
			return err
		}
		for _, con := range v.Contents {
			if con.Key == "" {
				continue
			}
			name := strings.TrimPrefix(con.Key, prefix)
			if name == "" {
				continue
			}
			fi := restic.FileInfo{
				Name: path.Base(name),
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
		for _, com := range v.CommonPrefixes {
			err := t.listv1(ctx, com, fn)
			if err != nil {
				return err
			}
		}
		isTruncated = v.IsTruncated // 是否还有数据
		marker = v.NextMarker       // 设置下次请求的起始 key
	}
	return nil
}

func (t *TxCos) List(ctx context.Context, ty restic.FileType, fn func(restic.FileInfo) error) error {
	prefix, _ := t.Basedir(ty)

	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	err := t.listv1(ctx, prefix, fn)
	if err != nil {
		return err
	}
	return ctx.Err()
}

func (t *TxCos) IsNotExist(err error) bool {
	if err == nil {
		return false
	}
	e, ok := err.(*cos.ErrorResponse)
	if !ok {
		return false
	}
	if e.Response != nil && e.Response.StatusCode == 404 {
		return true
	}
	return false
}

// IsAccessDenied returns true if the error is caused by Access Denied.
func (t *TxCos) IsAccessDenied(err error) bool {
	if err == nil {
		return false
	}
	if cosError, ok := err.(*cos.ErrorResponse); ok {
		if cosError.Response.StatusCode == 403 {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

func (t *TxCos) Delete(ctx context.Context) error {
	return backend.DefaultDelete(ctx, t)
}

// Join combines path components with slashes.
func (be *TxCos) Join(p ...string) string {
	return path.Join(p...)
}
