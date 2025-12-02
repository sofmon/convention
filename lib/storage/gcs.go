package storage

import (
	"context"
	"io"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	convCtx "github.com/sofmon/convention/lib/ctx"
)

func init() {
	RegisterProvider("gcs", newGCSProvider)
}

type gcsProvider struct {
	client *storage.Client
	bucket string
}

// newGCSProvider creates a GCS provider using the provided service account JSON credentials.
// The credentials parameter should contain the JSON key file content downloaded from GCP.
func newGCSProvider(bucket string, credentials []byte) (Provider, error) {
	client, err := storage.NewClient(
		context.Background(),
		option.WithCredentialsJSON(credentials),
	)
	if err != nil {
		return nil, err
	}
	return &gcsProvider{client: client, bucket: bucket}, nil
}

func (p *gcsProvider) Name() string {
	return "gcs"
}

func (p *gcsProvider) Save(ctx convCtx.Context, path string, data []byte) (err error) {
	ctx = ctx.WithScope("gcsProvider.Save", "path", path, "size", len(data))
	defer ctx.Exit(&err)

	w := p.client.Bucket(p.bucket).Object(path).NewWriter(context.Background())
	if _, err = w.Write(data); err != nil {
		return
	}
	err = w.Close()
	return
}

func (p *gcsProvider) Load(ctx convCtx.Context, path string) (data []byte, err error) {
	ctx = ctx.WithScope("gcsProvider.Load", "path", path)
	defer ctx.Exit(&err)

	r, err := p.client.Bucket(p.bucket).Object(path).NewReader(context.Background())
	if err != nil {
		return
	}
	defer r.Close()

	data, err = io.ReadAll(r)
	return
}

func (p *gcsProvider) Delete(ctx convCtx.Context, path string) (err error) {
	ctx = ctx.WithScope("gcsProvider.Delete", "path", path)
	defer ctx.Exit(&err)

	err = p.client.Bucket(p.bucket).Object(path).Delete(context.Background())
	if err == storage.ErrObjectNotExist {
		err = nil // idempotent delete
	}
	return
}

func (p *gcsProvider) Exists(ctx convCtx.Context, path string) (exists bool, err error) {
	ctx = ctx.WithScope("gcsProvider.Exists", "path", path)
	defer ctx.Exit(&err)

	_, err = p.client.Bucket(p.bucket).Object(path).Attrs(context.Background())
	if err == storage.ErrObjectNotExist {
		return false, nil
	}
	if err != nil {
		return
	}
	return true, nil
}
