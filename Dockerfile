FROM ubuntu:24.04

# Version arguments for easy updates
ARG ZELLIJ_VERSION=0.43.1
ARG HELIX_VERSION=25.07.1

# Install basic tools and dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        git \
        curl \
        wget \
        vim \
        ca-certificates \
        xz-utils \
        tar \
    && rm -rf /var/lib/apt/lists/*

# Install Zellij (Terminal Multiplexer)
ARG TARGETARCH
RUN ZELLIJ_ARCH=$([ "$TARGETARCH" = "arm64" ] && echo "aarch64" || echo "x86_64") && \
    wget -q "https://github.com/zellij-org/zellij/releases/download/v${ZELLIJ_VERSION}/zellij-${ZELLIJ_ARCH}-unknown-linux-musl.tar.gz" -O /tmp/zellij.tar.gz && \
    tar -xzf /tmp/zellij.tar.gz -C /usr/local/bin && \
    chmod +x /usr/local/bin/zellij && \
    rm /tmp/zellij.tar.gz

# Install Helix (Text Editor)
RUN HELIX_ARCH=$([ "$TARGETARCH" = "arm64" ] && echo "aarch64" || echo "x86_64") && \
    wget -q "https://github.com/helix-editor/helix/releases/download/${HELIX_VERSION}/helix-${HELIX_VERSION}-${HELIX_ARCH}-linux.tar.xz" -O /tmp/helix.tar.xz && \
    tar -xJf /tmp/helix.tar.xz -C /tmp && \
    mv /tmp/helix-${HELIX_VERSION}-${HELIX_ARCH}-linux/hx /usr/local/bin/ && \
    mkdir -p /usr/local/lib/helix && \
    mv /tmp/helix-${HELIX_VERSION}-${HELIX_ARCH}-linux/runtime /usr/local/lib/helix/ && \
    chmod +x /usr/local/bin/hx && \
    rm -rf /tmp/helix.tar.xz /tmp/helix-${HELIX_VERSION}-${HELIX_ARCH}-linux

# Setup Helix runtime environment
ENV HELIX_RUNTIME=/usr/local/lib/helix/runtime

# Note: Claude Code is now installed via init container
# This reduces image size and allows version updates without rebuilding

# Create workspace directory
RUN mkdir -p /workspace

# Set working directory
WORKDIR /workspace

# Keep container running
CMD ["sleep", "infinity"]
