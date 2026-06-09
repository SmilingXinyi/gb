#ifndef OSS_CGO_H
#define OSS_CGO_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/*
 * oss_free_string frees a C string previously returned by any oss_* function.
 * Callers MUST call this on every non-NULL return value to avoid memory leaks.
 */
void oss_free_string(char *s);

/*
 * oss_new_client creates a Storage client for the baidu provider.
 *
 * Parameters (all C strings, UTF-8, caller owns):
 *   endpoint   - BOS endpoint, e.g. "bj.bcebos.com" (empty string → derived from region)
 *   region     - region code, e.g. "bj" (used only when endpoint is empty)
 *   access_key - Baidu Cloud AK
 *   secret_key - Baidu Cloud SK
 *   token      - STS session token (empty string if not using STS)
 *   bucket     - default bucket name (empty string if not using a default)
 *   out_err    - if non-NULL, receives an error C string on failure (caller must free)
 *
 * Returns: opaque handle (> 0) on success, 0 on failure.
 */
int64_t oss_new_client(const char *endpoint,
                       const char *region,
                       const char *access_key,
                       const char *secret_key,
                       const char *token,
                       const char *bucket,
                       char      **out_err);

/*
 * oss_free_client destroys the client identified by handle and releases resources.
 * After this call the handle is invalid. Safe to call with handle=0 (no-op).
 */
void oss_free_client(int64_t handle);

/*
 * oss_put uploads a local file to OSS.
 *
 * meta_pairs: flat "key=value\n" pairs, e.g. "author=alice\nenv=prod\n"
 *             Pass empty string or NULL for no metadata.
 * content_type, storage_class, acl: empty string for provider defaults.
 *
 * Returns NULL on success, error string on failure (caller must free via oss_free_string).
 */
char *oss_put(int64_t     handle,
              const char *bucket,
              const char *key,
              const char *file_path,
              const char *content_type,
              const char *meta_pairs,
              const char *storage_class,
              const char *acl);

/*
 * oss_get downloads an object and writes it to dest_path (local file).
 * The destination file is created/truncated. Parent directory must exist.
 *
 * Returns NULL on success, error string on failure (caller must free via oss_free_string).
 */
char *oss_get(int64_t     handle,
              const char *bucket,
              const char *key,
              const char *dest_path);

/*
 * oss_delete deletes an object.
 *
 * Returns NULL on success, error string on failure (caller must free via oss_free_string).
 */
char *oss_delete(int64_t     handle,
                 const char *bucket,
                 const char *key);

/*
 * oss_stat returns object metadata as a JSON string via out_json.
 *
 * JSON shape:
 * {
 *   "key": "path/to/obj",
 *   "size": 1234,
 *   "content_type": "image/png",
 *   "etag": "abc123",
 *   "last_modified": "2024-01-02T15:04:05Z",
 *   "storage_class": "STANDARD",
 *   "metadata": {"author": "alice"}
 * }
 *
 * out_json: receives the JSON C string on success (caller must free via oss_free_string).
 * Returns NULL on success, error string on failure (caller must free via oss_free_string).
 */
char *oss_stat(int64_t     handle,
               const char *bucket,
               const char *key,
               char      **out_json);

/*
 * oss_list lists objects under a prefix and returns a JSON string via out_json.
 *
 * delimiter: pass empty string for no delimiter (flat list).
 * max_keys: 0 means use provider default (usually 1000).
 * continuation_token: empty string for the first page.
 *
 * JSON shape:
 * {
 *   "objects": [{"key":"a","size":10,"etag":"...","last_modified":"...","storage_class":"STANDARD"},...],
 *   "common_prefixes": ["dir/"],
 *   "next_token": "",
 *   "is_truncated": false
 * }
 *
 * out_json: receives the JSON C string on success (caller must free via oss_free_string).
 * Returns NULL on success, error string on failure (caller must free via oss_free_string).
 */
char *oss_list(int64_t     handle,
               const char *bucket,
               const char *prefix,
               const char *delimiter,
               int         max_keys,
               const char *continuation_token,
               char      **out_json);

/*
 * oss_sign_url generates a presigned URL.
 *
 * method: "GET" or "PUT"
 * expire_seconds: URL validity duration in seconds
 *
 * out_url: receives the URL C string on success (caller must free via oss_free_string).
 * Returns NULL on success, error string on failure (caller must free via oss_free_string).
 */
char *oss_sign_url(int64_t     handle,
                   const char *bucket,
                   const char *key,
                   const char *method,
                   int64_t     expire_seconds,
                   char      **out_url);

/*
 * oss_copy performs a server-side copy.
 *
 * Returns NULL on success, error string on failure (caller must free via oss_free_string).
 */
char *oss_copy(int64_t     handle,
               const char *src_bucket,
               const char *src_key,
               const char *dst_bucket,
               const char *dst_key);

#ifdef __cplusplus
}
#endif

#endif /* OSS_CGO_H */
