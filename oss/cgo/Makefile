OUT_DIR := ../../output/oss_cgo
LIB     := liboss

# Detect OS for shared library extension
UNAME := $(shell uname -s)
ifeq ($(UNAME),Darwin)
    SO_EXT := dylib
else
    SO_EXT := so
endif

SO_FILE := $(OUT_DIR)/$(LIB).$(SO_EXT)
H_FILE  := $(OUT_DIR)/$(LIB).h

.PHONY: all clean

all: $(SO_FILE)

$(OUT_DIR):
	mkdir -p $(OUT_DIR)

$(SO_FILE): $(OUT_DIR) oss_cgo.go handle.go json_helpers.go oss_cgo.h
	CGO_ENABLED=1 go build \
	  -buildmode=c-shared \
	  -o $(SO_FILE) \
	  .

clean:
	rm -rf $(OUT_DIR)
