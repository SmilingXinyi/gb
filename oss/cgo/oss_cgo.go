package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"unsafe"

	"github.com/SmilingXinyi/gb/oss"
	_ "github.com/SmilingXinyi/gb/oss/baidu" // registers baidu provider
)

// errStr converts a Go error to a C string allocated with C.CString.
// Returns nil for nil error. Caller owns the returned pointer.
func errStr(err error) *C.char {
	if err == nil {
		return nil
	}
	return C.CString(err.Error())
}

// goStringOrEmpty safely converts a potentially NULL *C.char to a Go string.
// C.GoString(nil) is undefined behavior; this helper avoids that.
func goStringOrEmpty(s *C.char) string {
	if s == nil {
		return ""
	}
	return C.GoString(s)
}

// parsePutMeta parses "key=value\nkey=value\n" into map[string]string.
// Only the first '=' in each line is treated as separator.
func parsePutMeta(pairs string) map[string]string {
	pairs = strings.TrimRight(pairs, "\n")
	if pairs == "" {
		return nil
	}
	result := make(map[string]string)
	for _, line := range strings.Split(pairs, "\n") {
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		result[line[:idx]] = line[idx+1:]
	}
	return result
}

//export oss_free_string
func oss_free_string(s *C.char) {
	C.free(unsafe.Pointer(s))
}

//export oss_new_client
func oss_new_client(endpoint, region, accessKey, secretKey, token, bucket *C.char, outErr **C.char) C.int64_t {
	cfg := oss.Config{
		Endpoint:  C.GoString(endpoint),
		Region:    C.GoString(region),
		AccessKey: C.GoString(accessKey),
		SecretKey: C.GoString(secretKey),
		Token:     C.GoString(token),
		Bucket:    C.GoString(bucket),
	}
	storage, err := oss.New(oss.ProviderBaidu, cfg)
	if err != nil {
		if outErr != nil {
			*outErr = C.CString(err.Error())
		}
		return 0
	}
	return C.int64_t(storeClient(storage))
}

//export oss_free_client
func oss_free_client(handle C.int64_t) {
	deleteClient(int64(handle))
}

//export oss_put
func oss_put(handle C.int64_t, bucket, key, filePath, contentType, metaPairs, storageClass, acl *C.char) *C.char {
	storage, ok := loadClient(int64(handle))
	if !ok {
		return C.CString("oss_put: invalid handle")
	}
	f, err := os.Open(C.GoString(filePath))
	if err != nil {
		return errStr(fmt.Errorf("oss_put: open file: %w", err))
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return errStr(fmt.Errorf("oss_put: stat file: %w", err))
	}

	opts := &oss.PutOptions{
		ContentType:  C.GoString(contentType),
		Metadata:     parsePutMeta(goStringOrEmpty(metaPairs)),
		StorageClass: C.GoString(storageClass),
		ACL:          C.GoString(acl),
	}

	err = storage.Put(context.Background(), C.GoString(bucket), C.GoString(key), f, stat.Size(), opts)
	return errStr(err)
}

//export oss_get
func oss_get(handle C.int64_t, bucket, key, destPath *C.char) *C.char {
	storage, ok := loadClient(int64(handle))
	if !ok {
		return C.CString("oss_get: invalid handle")
	}
	rc, err := storage.Get(context.Background(), C.GoString(bucket), C.GoString(key))
	if err != nil {
		return errStr(err)
	}
	defer rc.Close()

	dest, err := os.Create(C.GoString(destPath))
	if err != nil {
		return errStr(fmt.Errorf("oss_get: create dest: %w", err))
	}

	if _, err = io.Copy(dest, rc); err != nil {
		dest.Close()
		os.Remove(C.GoString(destPath))
		return errStr(fmt.Errorf("oss_get: write dest: %w", err))
	}
	dest.Close()
	return nil
}

//export oss_delete
func oss_delete(handle C.int64_t, bucket, key *C.char) *C.char {
	storage, ok := loadClient(int64(handle))
	if !ok {
		return C.CString("oss_delete: invalid handle")
	}
	err := storage.Delete(context.Background(), C.GoString(bucket), C.GoString(key))
	return errStr(err)
}

//export oss_stat
func oss_stat(handle C.int64_t, bucket, key *C.char, outJSON **C.char) *C.char {
	storage, ok := loadClient(int64(handle))
	if !ok {
		return C.CString("oss_stat: invalid handle")
	}
	meta, err := storage.Stat(context.Background(), C.GoString(bucket), C.GoString(key))
	if err != nil {
		return errStr(err)
	}
	jsonStr, err := marshalObjectMeta(meta)
	if err != nil {
		return errStr(fmt.Errorf("oss_stat: marshal: %w", err))
	}
	*outJSON = C.CString(jsonStr)
	return nil
}

//export oss_list
func oss_list(handle C.int64_t, bucket, prefix, delimiter *C.char, maxKeys C.int, continuationToken *C.char, outJSON **C.char) *C.char {
	storage, ok := loadClient(int64(handle))
	if !ok {
		return C.CString("oss_list: invalid handle")
	}
	opts := &oss.ListOptions{
		Delimiter:         C.GoString(delimiter),
		MaxKeys:           int(maxKeys),
		ContinuationToken: C.GoString(continuationToken),
	}
	result, err := storage.List(context.Background(), C.GoString(bucket), C.GoString(prefix), opts)
	if err != nil {
		return errStr(err)
	}
	jsonStr, err := marshalListResult(result)
	if err != nil {
		return errStr(fmt.Errorf("oss_list: marshal: %w", err))
	}
	*outJSON = C.CString(jsonStr)
	return nil
}

//export oss_sign_url
func oss_sign_url(handle C.int64_t, bucket, key, method *C.char, expireSeconds C.int64_t, outURL **C.char) *C.char {
	storage, ok := loadClient(int64(handle))
	if !ok {
		return C.CString("oss_sign_url: invalid handle")
	}
	url, err := storage.SignURL(context.Background(), C.GoString(bucket), C.GoString(key), C.GoString(method), int64(expireSeconds))
	if err != nil {
		return errStr(err)
	}
	*outURL = C.CString(url)
	return nil
}

//export oss_copy
func oss_copy(handle C.int64_t, srcBucket, srcKey, dstBucket, dstKey *C.char) *C.char {
	storage, ok := loadClient(int64(handle))
	if !ok {
		return C.CString("oss_copy: invalid handle")
	}
	err := storage.Copy(context.Background(), C.GoString(srcBucket), C.GoString(srcKey), C.GoString(dstBucket), C.GoString(dstKey))
	return errStr(err)
}

func main() {} // required by c-shared build mode
