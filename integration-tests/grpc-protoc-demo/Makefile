# gRPC Protobuf Demo - Native protoc (no buf)
# This project demonstrates using spec-forge with native protoc

PROTO_DIR := proto
GEN_DIR := gen
OPENAPI_OUT := $(GEN_DIR)/openapi

# Import paths (can be extended via IMPORT_PATHS environment variable)
IMPORT_PATHS := $(PROTO_DIR)
# Additional import paths from environment
ifdef EXTRA_IMPORT_PATHS
IMPORT_PATHS := $(IMPORT_PATHS) $(EXTRA_IMPORT_PATHS)
endif

# Build protoc import flags
PROTOC_IMPORT_FLAGS := $(foreach path,$(IMPORT_PATHS),-I$(path))

.PHONY: all clean proto openapi

all: proto openapi

# Generate Go code from proto files
proto:
	@echo "Generating Go code from proto files..."
	@mkdir -p $(GEN_DIR)
	protoc $(PROTOC_IMPORT_FLAGS) \
		--go_out=$(GEN_DIR) \
		--go_opt=paths=source_relative \
		$(PROTO_DIR)/*.proto

# Generate OpenAPI spec from proto files using protoc-gen-connect-openapi
openapi:
	@echo "Generating OpenAPI spec from proto files..."
	@mkdir -p $(OPENAPI_OUT)
	protoc $(PROTOC_IMPORT_FLAGS) \
		--connect-openapi_out=$(OPENAPI_OUT) \
		--connect-openapi_opt=features=google.api.http \
		$(PROTO_DIR)/user.proto

# Clean generated files
clean:
	@rm -rf $(GEN_DIR)

# Check if required tools are installed
check-tools:
	@echo "Checking required tools..."
	@which protoc > /dev/null || (echo "ERROR: protoc not found. Install from https://github.com/protocolbuffers/protobuf/releases" && exit 1)
	@which protoc-gen-go > /dev/null || (echo "ERROR: protoc-gen-go not found. Run: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest" && exit 1)
	@which protoc-gen-connect-openapi > /dev/null || (echo "ERROR: protoc-gen-connect-openapi not found. Run: go install github.com/sudorandom/protoc-gen-connect-openapi@latest" && exit 1)
	@echo "All required tools are installed!"

# Install required Go tools
install-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install github.com/sudorandom/protoc-gen-connect-openapi@latest
