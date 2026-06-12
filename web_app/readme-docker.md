# Docker Notes

This project can be run in Docker for deployment-style testing, but day-to-day development is usually faster when only dependencies such as PostgreSQL run in Docker and the app services run from local terminals.

## Full Docker Deployment

Start the full stack:

```powershell
docker compose up -d
```

The `-d` flag means detached mode. Containers run in the background and the terminal prompt returns immediately.

If application code changed and the images need to be rebuilt:

```powershell
docker compose up -d --build
```

Equivalent explicit form:

```powershell
docker compose build
docker compose up -d
```

Check container status:

```powershell
docker compose ps
```

Follow logs:

```powershell
docker compose logs -f
```

Stop and remove the Compose containers:

```powershell
docker compose down
```

## What Gets Built

The current Compose setup builds local images through Docker Desktop:

- `postgres` is pulled from `postgres:16-alpine`.
- `ml_service` is built from `ml_service/Dockerfile`.
- `server` is built from `server/Dockerfile`.
- The frontend is built inside the `server` image build using `node:22-alpine`.
- The Go backend is built inside the `server` image build using `golang:1.22-alpine`.

The source files are on Windows, but the build commands run inside Docker's Linux environment. A Go installation on Windows does not provide Go modules or compiler access inside the Docker build container.

## Local Development

For daily development, it is often easier to run only PostgreSQL in Docker:

```powershell
docker compose up -d postgres
```

Then run the backend locally:

```powershell
cd D:\yxu\mesh3d\web_app\server
go run .
```

Run the frontend locally:

```powershell
cd D:\yxu\mesh3d\web_app\client
pnpm install
pnpm dev
```

With this workflow, frontend changes usually hot reload through the dev server. Go backend changes can be applied by restarting `go run .`, unless a Go hot-reload tool is added later.

If the backend runs on Windows and PostgreSQL runs in Docker, the database host should be `localhost`, not the Compose service name `postgres`.

Example local database URL:

```text
postgres://mesh3d:mesh3d_dev@localhost:5432/mesh3d?sslmode=disable
```

If local code cannot connect to PostgreSQL, make sure the Compose file publishes the database port:

```yaml
postgres:
  ports:
    - "5432:5432"
```

## Rebuilding After Code Changes

For Docker deployment, code is copied into the image during build. Changing local files does not automatically change an already-built container.

After code changes, redeploy with:

```powershell
docker compose up -d --build
```

If only `docker-compose.yml` settings changed, such as environment variables, ports, or volumes, this is often enough:

```powershell
docker compose up -d
```

Compose will recreate affected containers when needed.

## Docker Network Notes

Errors such as these usually indicate Docker Desktop networking or registry access issues:

```text
TLS handshake timeout
failed to fetch anonymous token
failed to solve: process "/bin/sh -c go mod download" did not complete successfully
```

This can happen even when a VPN works for normal Windows apps, because Docker Desktop uses its own Linux VM/network path.

The Go dependency download step runs inside the Docker build container:

```dockerfile
RUN go mod download
```

If this fails, Docker's build environment cannot reach the Go module proxy reliably. One workaround is to set an alternate Go proxy before `go mod download`:

```dockerfile
ENV GOPROXY=https://goproxy.cn,direct
```

Pulling the Go base image only provides the compiler image. Project dependencies still need to be downloaded by `go mod download`, unless dependencies are vendored or the build is changed to copy a prebuilt Linux binary.
