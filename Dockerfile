FROM debian:bookworm-slim

WORKDIR /app

COPY app .

ENTRYPOINT ["./app"] 