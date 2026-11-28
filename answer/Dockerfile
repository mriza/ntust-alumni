# Stage 1: Extract base Apache Answer binary, entrypoint, and data layout
FROM docker.io/apache/answer:latest AS answer-builder

# Stage 2: Compile Apache Answer with connector-basic plugin
FROM docker.io/library/golang:1.25-alpine AS golang-builder

COPY --from=answer-builder /usr/bin/answer /usr/bin/answer

RUN apk --no-cache add \
    build-base \
    git \
    bash \
    nodejs \
    npm && \
    npm install -g pnpm@10.7.0

RUN answer build \
    --with github.com/apache/answer-plugins/connector-basic \
    --output /usr/bin/new_answer

# Stage 3: Final runtime preserving official /entrypoint.sh and /data layout
FROM docker.io/apache/answer:latest

COPY --from=golang-builder /usr/bin/new_answer /usr/bin/answer
