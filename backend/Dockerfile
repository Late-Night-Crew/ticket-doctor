# Production image for the API (deployed to Render from release tags; see RELEASING.md).
# Same two-stage shape as the School Management System image: build with the Go
# toolchain, then copy only the binary into a small Alpine image.

FROM golang:1.26-alpine AS build
WORKDIR /src
# Download modules first so this layer is cached until go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# The release workflow passes the tag (e.g. v0.2.0); /api/health reports it.
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X ticketdoctor/cmd.version=${VERSION}" -o /out/ticketdoctor .

FROM alpine:3.24
ARG VERSION=dev
# Links the image to the repository on GitHub, and records its version.
LABEL org.opencontainers.image.source="https://github.com/Late-Night-Crew/ticket-doctor" \
      org.opencontainers.image.description="Ticket Doctor API" \
      org.opencontainers.image.version="${VERSION}"
RUN apk add --no-cache ca-certificates tzdata && adduser -D -h /app app
WORKDIR /app
COPY --from=build /out/ticketdoctor /app/ticketdoctor
COPY deploy/start.sh /app/start.sh
RUN chmod +x /app/start.sh
# Never run as root.
USER app
# Inside a container the API must listen on all interfaces. Everything else
# (database, allowed origins) comes from the host's environment variables.
ENV TD_HOST=0.0.0.0 \
    TD_ENVIRONMENT=PRODUCTION \
    TD_LOG_FORMAT=json
EXPOSE 8080
CMD ["/app/start.sh"]
