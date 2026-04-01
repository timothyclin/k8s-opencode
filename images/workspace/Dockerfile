# The stock ghcr.io/anomalyco/opencode image is a minimal Alpine/musl build
# with only libgcc, libstdc++, and ripgrep. This image provides a full
# developer environment on glibc (Debian) with git, compilers, and common
# CLI tools so hosted OpenCode sessions can actually build, test, and commit.
#
ARG OPENCODE_IMAGE=ghcr.io/anomalyco/opencode:latest

# ---------------------------------------------------------------------------
# Stage 1: Extract the opencode binary from the stock image
# ---------------------------------------------------------------------------
FROM ${OPENCODE_IMAGE} AS opencode-bin

# ---------------------------------------------------------------------------
# Stage 2: Developer-ready base
# ---------------------------------------------------------------------------
FROM debian:bookworm-slim

LABEL org.opencontainers.image.title="opencode-workspace" \
      org.opencontainers.image.description="Developer-ready OpenCode image with git, compilers, and common CLI tools" \
      org.opencontainers.image.source="https://github.com/timothyclin/k8s-opencode"

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    # Version control
    git \
    git-lfs \
    openssh-client \
    # Build essentials
    build-essential \
    pkg-config \
    # Runtime / scripting
    python3 \
    python3-venv \
    # Network / download
    curl \
    wget \
    ca-certificates \
    # Shell & terminal
    bash \
    tmux \
    less \
    procps \
    # Search & file tools
    ripgrep \
    jq \
    tree \
    unzip \
    zip \
    xz-utils \
    # Misc utilities
    sudo \
    lsof \
    && rm -rf /var/lib/apt/lists/*

COPY --from=opencode-bin /usr/local/bin/opencode /usr/local/bin/opencode
RUN chmod +x /usr/local/bin/opencode && opencode --version

WORKDIR /workspace

ENTRYPOINT ["opencode"]
