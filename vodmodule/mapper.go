package vodmodule

import (
	"context"
	"errors"
	"net/http"
	"path"
	"regexp"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
)

const maxTries = 5

// retryDelay is the wait before the first retry; it doubles on each retry.
var retryDelay = 100 * time.Millisecond

// Mapper provides the ability of mapping objects on a GCS bucket in the format
// expected by nginx-vod-module.
type Mapper struct {
	bucket *storage.BucketHandle
}

// NewMapper returns a mapper that will map content for prefix in the given
// BucketHandle.
func NewMapper(bucket *storage.BucketHandle) *Mapper {
	return &Mapper{bucket: bucket}
}

// MapOptions represents the set of options that can be passed to Map.
type MapOptions struct {
	Prefix string

	// Optional regexp that is used to filter the list of objects.
	Filter *regexp.Regexp
}

// Map returns a Mapping object with the list of objects that match the given
// prefix. It supports a regular expression that is used to further filter (for
// example, if the caller only wants to return objects that with the ".mp4"
// extension).
func (m *Mapper) Map(ctx context.Context, opts MapOptions) (Mapping, error) {
	var err error
	r := Mapping{}
	r.Sequences, err = m.getSequences(ctx, opts.Prefix, opts.Filter)
	return r, err
}

func (m *Mapper) getSequences(ctx context.Context, prefix string, filter *regexp.Regexp) ([]Sequence, error) {
	var err error
	delay := retryDelay
	for i := 0; i < maxTries; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, err
			case <-time.After(delay):
			}
			delay *= 2
		}
		iter := m.bucket.Objects(ctx, &storage.Query{
			Prefix:    prefix,
			Delimiter: "/",
		})
		seqs := []Sequence{}
		var obj *storage.ObjectAttrs
		obj, err = iter.Next()
		for ; err == nil; obj, err = iter.Next() {
			// With a delimiter, "subdirectories" are listed as entries
			// that only have Prefix set.
			if obj.Name == "" {
				continue
			}
			filename := path.Base(obj.Name)
			if filter == nil || filter.MatchString(filename) {
				seqs = append(seqs, Sequence{
					Clips: []Clip{{Type: "source", Path: "/" + obj.Bucket + "/" + obj.Name}},
				})
			}
		}
		if err == iterator.Done {
			return seqs, nil
		}
		if ctx.Err() != nil || !retryable(err) {
			return nil, err
		}
	}
	return nil, err
}

// retryable reports whether a listing error may succeed on a later attempt.
func retryable(err error) bool {
	if errors.Is(err, storage.ErrBucketNotExist) {
		return false
	}
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		return apiErr.Code == http.StatusTooManyRequests || apiErr.Code >= http.StatusInternalServerError
	}
	return true
}
