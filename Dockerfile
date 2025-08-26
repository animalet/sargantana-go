FROM node:20-alpine AS frontend-builder
WORKDIR /ui
# Create a default Vite site
RUN npm create vite@latest . -- --template vanilla
RUN npm install
RUN npm run build
RUN apk add --no-cache ca-certificates

FROM golang:1.25-alpine3.22 AS backend-builder
WORKDIR /app
COPY . ./
ARG TARGETPLATFORM
ARG TARGETARCH
ARG TARGETOS

# Use the -trimpath flag to remove file system paths from the compiled binary
ENV CGO_ENABLED=0
ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}
RUN echo "I'm building for $TARGETPLATFORM on $TARGETARCH architecture for $GOOS OS"

RUN go mod download
RUN go build -trimpath -ldflags="-s -w" -o server
RUN chmod +x /app/server

FROM scratch
COPY --from=backend-builder /app/server /app/server
COPY --from=backend-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=frontend-builder /ui/dist /app/frontend
EXPOSE 8080
ENTRYPOINT ["/app/server"]