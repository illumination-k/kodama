FROM ubuntu:22.04

# Install git and basic tools
RUN apt-get update && \
    apt-get install -y \
    git \
    curl \
    wget \
    vim \
    && rm -rf /var/lib/apt/lists/*

# Create workspace directory
RUN mkdir -p /workspace

# Set working directory
WORKDIR /workspace

# Keep container running
CMD ["sleep", "infinity"]
