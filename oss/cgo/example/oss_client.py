"""
oss_client.py — Python ctypes wrapper for the oss CGo shared library.

Loads liboss.so (or liboss.dylib on macOS) and exposes a high-level OssClient class
that wraps all 7 Storage operations: Put, Get, Delete, Stat, List, SignURL, Copy.

Quick start:
    from oss_client import OssClient

    with OssClient.new(
        lib_path="path/to/liboss.dylib",
        access_key="your-ak",
        secret_key="your-sk",
        region="bj",
        bucket="my-bucket",
    ) as client:
        client.put("my-bucket", "data/hello.txt", "/tmp/hello.txt")
        client.get("my-bucket", "data/hello.txt", "/tmp/downloaded.txt")
        meta = client.stat("my-bucket", "data/hello.txt")
        result = client.list("my-bucket", "data/")
        url = client.sign_url("my-bucket", "data/hello.txt", "GET", 3600)
        client.copy("my-bucket", "data/hello.txt", "my-bucket", "backup/hello.txt")
        client.delete("my-bucket", "data/hello.txt")

Memory management notes:
  - All C strings returned by the library are freed automatically by this wrapper.
  - Calling oss_free_string is handled internally; callers never need to manage memory.

Error handling:
  - All methods raise RuntimeError on failure with the error message from the library.
"""

from __future__ import annotations

import ctypes
import json
import os
import platform
import sys
import tempfile


def _load_lib(lib_path: str) -> ctypes.CDLL:
    """Load the shared library and configure all function signatures."""
    lib = ctypes.CDLL(lib_path)

    # void oss_free_string(char *s)
    lib.oss_free_string.restype = None
    lib.oss_free_string.argtypes = [ctypes.c_char_p]

    # int64_t oss_new_client(endpoint, region, ak, sk, token, bucket *char, out_err **char)
    lib.oss_new_client.restype = ctypes.c_int64
    lib.oss_new_client.argtypes = [
        ctypes.c_char_p,                  # endpoint
        ctypes.c_char_p,                  # region
        ctypes.c_char_p,                  # access_key
        ctypes.c_char_p,                  # secret_key
        ctypes.c_char_p,                  # token
        ctypes.c_char_p,                  # bucket
        ctypes.POINTER(ctypes.c_char_p),  # out_err
    ]

    # void oss_free_client(int64_t handle)
    lib.oss_free_client.restype = None
    lib.oss_free_client.argtypes = [ctypes.c_int64]

    # char* oss_put(handle, bucket, key, file_path, content_type, meta_pairs, storage_class, acl)
    lib.oss_put.restype = ctypes.c_char_p
    lib.oss_put.argtypes = [
        ctypes.c_int64,
        ctypes.c_char_p, ctypes.c_char_p,  # bucket, key
        ctypes.c_char_p,                    # file_path
        ctypes.c_char_p,                    # content_type
        ctypes.c_char_p,                    # meta_pairs ("k=v\n...")
        ctypes.c_char_p,                    # storage_class
        ctypes.c_char_p,                    # acl
    ]

    # char* oss_get(handle, bucket, key, dest_path)
    lib.oss_get.restype = ctypes.c_char_p
    lib.oss_get.argtypes = [
        ctypes.c_int64,
        ctypes.c_char_p, ctypes.c_char_p,
        ctypes.c_char_p,
    ]

    # char* oss_delete(handle, bucket, key)
    lib.oss_delete.restype = ctypes.c_char_p
    lib.oss_delete.argtypes = [
        ctypes.c_int64,
        ctypes.c_char_p, ctypes.c_char_p,
    ]

    # char* oss_stat(handle, bucket, key, out_json **char)
    lib.oss_stat.restype = ctypes.c_char_p
    lib.oss_stat.argtypes = [
        ctypes.c_int64,
        ctypes.c_char_p, ctypes.c_char_p,
        ctypes.POINTER(ctypes.c_char_p),
    ]

    # char* oss_list(handle, bucket, prefix, delimiter, max_keys, continuation_token, out_json **char)
    lib.oss_list.restype = ctypes.c_char_p
    lib.oss_list.argtypes = [
        ctypes.c_int64,
        ctypes.c_char_p, ctypes.c_char_p,  # bucket, prefix
        ctypes.c_char_p,                    # delimiter
        ctypes.c_int,                       # max_keys
        ctypes.c_char_p,                    # continuation_token
        ctypes.POINTER(ctypes.c_char_p),    # out_json
    ]

    # char* oss_sign_url(handle, bucket, key, method, expire_seconds, out_url **char)
    lib.oss_sign_url.restype = ctypes.c_char_p
    lib.oss_sign_url.argtypes = [
        ctypes.c_int64,
        ctypes.c_char_p, ctypes.c_char_p,
        ctypes.c_char_p,
        ctypes.c_int64,
        ctypes.POINTER(ctypes.c_char_p),
    ]

    # char* oss_copy(handle, src_bucket, src_key, dst_bucket, dst_key)
    lib.oss_copy.restype = ctypes.c_char_p
    lib.oss_copy.argtypes = [
        ctypes.c_int64,
        ctypes.c_char_p, ctypes.c_char_p,
        ctypes.c_char_p, ctypes.c_char_p,
    ]

    return lib


def _enc(s: str | None) -> bytes:
    """Encode a Python string (or None/empty) to UTF-8 bytes for ctypes."""
    if not s:
        return b""
    return s.encode("utf-8")


def _check_err(lib: ctypes.CDLL, raw: bytes | None) -> None:
    """Raise RuntimeError if raw is a non-None error string, then free it."""
    if raw is None:
        return
    msg = raw.decode("utf-8", errors="replace")
    # restype=c_char_p causes ctypes to copy bytes; we must still free the C memory.
    lib.oss_free_string(raw)
    raise RuntimeError(msg)


def _free_and_decode(lib: ctypes.CDLL, ptr: ctypes.c_char_p) -> str:
    """Decode a c_char_p value and free the underlying C memory."""
    value = ptr.value
    if value is not None:
        lib.oss_free_string(value)
    return (value or b"").decode("utf-8", errors="replace")


class OssClient:
    """High-level Python wrapper around the oss CGo shared library."""

    def __init__(self, lib: ctypes.CDLL, handle: int) -> None:
        self._lib = lib
        self._handle = handle

    @classmethod
    def new(
        cls,
        lib_path: str,
        access_key: str,
        secret_key: str,
        *,
        endpoint: str = "",
        region: str = "bj",
        token: str = "",
        bucket: str = "",
    ) -> "OssClient":
        """
        Create a new OssClient for the baidu provider.

        Args:
            lib_path:   Path to liboss.so / liboss.dylib.
            access_key: Baidu Cloud AK.
            secret_key: Baidu Cloud SK.
            endpoint:   BOS endpoint override (e.g. "bj.bcebos.com"). Optional.
            region:     Region code (e.g. "bj"). Used when endpoint is empty.
            token:      STS session token. Optional.
            bucket:     Default bucket name. Optional.
        """
        lib = _load_lib(lib_path)
        out_err = ctypes.c_char_p(None)
        handle = lib.oss_new_client(
            _enc(endpoint),
            _enc(region),
            _enc(access_key),
            _enc(secret_key),
            _enc(token),
            _enc(bucket),
            ctypes.byref(out_err),
        )
        if handle == 0:
            err_msg = ""
            if out_err.value:
                err_msg = out_err.value.decode("utf-8", errors="replace")
                lib.oss_free_string(out_err.value)
            raise RuntimeError(f"oss_new_client failed: {err_msg or 'unknown error'}")
        return cls(lib, handle)

    def close(self) -> None:
        """Destroy the client and release resources."""
        if self._handle != 0:
            self._lib.oss_free_client(self._handle)
            self._handle = 0

    def __enter__(self) -> "OssClient":
        return self

    def __exit__(self, *_) -> None:
        self.close()

    def put(
        self,
        bucket: str,
        key: str,
        file_path: str,
        *,
        content_type: str = "",
        metadata: dict[str, str] | None = None,
        storage_class: str = "",
        acl: str = "",
    ) -> None:
        """Upload a local file to OSS.

        Args:
            bucket:        Target bucket name. Empty string uses the default bucket.
            key:           Object key (path in the bucket).
            file_path:     Absolute or relative path to the local file to upload.
            content_type:  MIME type. Empty string lets the provider infer.
            metadata:      Custom key-value metadata. Keys/values must not contain '=' or newlines.
            storage_class: e.g. "STANDARD", "COLD". Empty string uses provider default.
            acl:           e.g. "private", "public-read". Empty string uses provider default.
        """
        meta_str = "".join(f"{k}={v}\n" for k, v in (metadata or {}).items())
        err = self._lib.oss_put(
            self._handle,
            _enc(bucket), _enc(key),
            _enc(file_path),
            _enc(content_type),
            _enc(meta_str),
            _enc(storage_class),
            _enc(acl),
        )
        _check_err(self._lib, err)

    def get(self, bucket: str, key: str, dest_path: str) -> None:
        """Download an object to a local file.

        Args:
            bucket:    Source bucket name.
            key:       Object key.
            dest_path: Local file path to write. Parent directory must exist.
        """
        err = self._lib.oss_get(
            self._handle,
            _enc(bucket), _enc(key),
            _enc(dest_path),
        )
        _check_err(self._lib, err)

    def delete(self, bucket: str, key: str) -> None:
        """Delete an object."""
        err = self._lib.oss_delete(
            self._handle,
            _enc(bucket), _enc(key),
        )
        _check_err(self._lib, err)

    def stat(self, bucket: str, key: str) -> dict:
        """Return object metadata as a Python dict.

        Returns a dict with keys:
            key, size, content_type, etag, last_modified (RFC3339),
            storage_class, metadata (dict[str, str])
        """
        out_json = ctypes.c_char_p(None)
        err = self._lib.oss_stat(
            self._handle,
            _enc(bucket), _enc(key),
            ctypes.byref(out_json),
        )
        _check_err(self._lib, err)
        raw = out_json.value
        if raw:
            self._lib.oss_free_string(raw)
        return json.loads(raw or b"{}")

    def list(
        self,
        bucket: str,
        prefix: str = "",
        *,
        delimiter: str = "",
        max_keys: int = 0,
        continuation_token: str = "",
    ) -> dict:
        """List objects under a prefix.

        Returns a dict with keys:
            objects (list of dicts), common_prefixes (list of str),
            next_token (str), is_truncated (bool)

        Each object dict has: key, size, etag, last_modified, storage_class.
        """
        out_json = ctypes.c_char_p(None)
        err = self._lib.oss_list(
            self._handle,
            _enc(bucket), _enc(prefix),
            _enc(delimiter),
            ctypes.c_int(max_keys),
            _enc(continuation_token),
            ctypes.byref(out_json),
        )
        _check_err(self._lib, err)
        raw = out_json.value
        if raw:
            self._lib.oss_free_string(raw)
        return json.loads(raw or b"{}")

    def sign_url(
        self,
        bucket: str,
        key: str,
        method: str = "GET",
        expire_seconds: int = 3600,
    ) -> str:
        """Generate a presigned URL.

        Args:
            method:         "GET" or "PUT".
            expire_seconds: URL validity duration in seconds.

        Returns the presigned URL as a string.
        """
        out_url = ctypes.c_char_p(None)
        err = self._lib.oss_sign_url(
            self._handle,
            _enc(bucket), _enc(key),
            _enc(method),
            ctypes.c_int64(expire_seconds),
            ctypes.byref(out_url),
        )
        _check_err(self._lib, err)
        raw = out_url.value
        url = (raw or b"").decode("utf-8")
        if raw:
            self._lib.oss_free_string(raw)
        return url

    def copy(
        self,
        src_bucket: str,
        src_key: str,
        dst_bucket: str,
        dst_key: str,
    ) -> None:
        """Perform a server-side copy."""
        err = self._lib.oss_copy(
            self._handle,
            _enc(src_bucket), _enc(src_key),
            _enc(dst_bucket), _enc(dst_key),
        )
        _check_err(self._lib, err)


# ---------------------------------------------------------------------------
# Demo / smoke test
# ---------------------------------------------------------------------------
if __name__ == "__main__":
    # Resolve lib path: prefer OSS_LIB_PATH env, then look next to this script.
    _script_dir = os.path.dirname(os.path.abspath(__file__))
    _default_lib = os.path.join(
        _script_dir,
        "..", "..", "..", "output", "oss_cgo",
        "liboss.dylib" if platform.system() == "Darwin" else "liboss.so",
    )
    LIB = os.getenv("OSS_LIB_PATH", _default_lib)
    AK  = os.getenv("BAIDU_OSS_AK", "")
    SK  = os.getenv("BAIDU_OSS_SK", "")
    BKT = os.getenv("BAIDU_OSS_BUCKET", "")

    if not AK or not SK or not BKT:
        print(
            "Usage: BAIDU_OSS_AK=<ak> BAIDU_OSS_SK=<sk> BAIDU_OSS_BUCKET=<bucket> "
            "[OSS_LIB_PATH=<path/to/liboss.so>] python oss_client.py",
            file=sys.stderr,
        )
        sys.exit(1)

    print(f"Loading library: {LIB}")
    with OssClient.new(LIB, AK, SK, region="bj", bucket=BKT) as client:
        key        = "cgo-test/hello.txt"
        copy_key   = "cgo-test/hello_copy.txt"

        # --- Put ---
        with tempfile.NamedTemporaryFile(delete=False, suffix=".txt") as f:
            f.write(b"hello from python ctypes")
            tmp_path = f.name
        client.put(
            BKT, key, tmp_path,
            content_type="text/plain",
            metadata={"author": "python", "env": "test"},
        )
        print(f"[put]      {key} OK")

        # --- Stat ---
        meta = client.stat(BKT, key)
        print(f"[stat]     size={meta['size']} content_type={meta['content_type']}")
        print(f"           metadata={meta['metadata']}")

        # --- Get ---
        dl_path = tmp_path + ".dl"
        client.get(BKT, key, dl_path)
        content = open(dl_path, "rb").read()
        print(f"[get]      content={content!r}")

        # --- List ---
        result = client.list(BKT, "cgo-test/", delimiter="/")
        print(f"[list]     {len(result['objects'])} object(s), prefixes={result['common_prefixes']}")

        # --- Sign URL ---
        url = client.sign_url(BKT, key, "GET", 3600)
        print(f"[sign_url] {url[:80]}...")

        # --- Copy ---
        client.copy(BKT, key, BKT, copy_key)
        print(f"[copy]     {key} → {copy_key} OK")

        # --- Delete ---
        client.delete(BKT, key)
        client.delete(BKT, copy_key)
        print(f"[delete]   {key}, {copy_key} OK")

        os.unlink(tmp_path)
        os.unlink(dl_path)

    print("All operations completed successfully.")
