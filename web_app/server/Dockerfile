# syntax=docker/dockerfile:1.7

# Build the frontend assets.
FROM node:22-alpine AS client-build
WORKDIR /src/client

RUN corepack enable
# Install dependencies first so Docker can cache this layer.
COPY client/package.json client/pnpm-lock.yaml client/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

COPY client/ ./
RUN pnpm build

# ------------------------------------------------------------- #
# Build the Go server binary.
FROM golang:1.22-alpine AS server-build
WORKDIR /src/server

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ ./
# Create a small Linux binary for the final image.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/mesh3d-server .

# Keep the runtime image small.
FROM alpine:3.20
WORKDIR /app/server

# Run the app as a non-root user.
RUN addgroup -S mesh3d && adduser -S -G mesh3d mesh3d

# Copy only the built server and client files.
COPY --from=server-build /out/mesh3d-server /app/server/mesh3d-server
COPY --from=client-build /src/client/dist /app/client/dist

# Default production settings.
ENV MESH3D_ENV=production
ENV MESH3D_ADDR=:8080
ENV MESH3D_CLIENT_DIR=/app/client/dist

RUN chown -R mesh3d:mesh3d /app

USER mesh3d
EXPOSE 8080
# Start the server.
CMD ["/app/server/mesh3d-server"]
