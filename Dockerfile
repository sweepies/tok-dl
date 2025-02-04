FROM gcr.io/distroless/static-debian12

WORKDIR /app

ENV TOKDL_CACHE_DIR="."

ENTRYPOINT ["/tok-dl"]
COPY tok-dl /