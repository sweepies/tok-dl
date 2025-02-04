FROM gcr.io/distroless/static-debian12
LABEL org.opencontainers.image.source https://github.com/sweepies/tok-dl

WORKDIR /app

ENV TOKDL_CACHE_DIR="."

ENTRYPOINT ["/tok-dl"]
COPY tok-dl /

