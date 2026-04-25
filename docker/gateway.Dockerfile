FROM node:20-bookworm AS frontend-build

WORKDIR /src

COPY frontend/package.json frontend/package-lock.json /src/frontend/
RUN npm ci --prefix /src/frontend

COPY frontend /src/frontend
COPY locales /src/locales

RUN npm run build --prefix /src/frontend

FROM golang:1.22-bookworm AS gateway-build

WORKDIR /src/gateway

COPY gateway/go.mod ./
RUN go mod download

COPY gateway /src/gateway

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/go-mirofish-gateway ./cmd/mirofish-gateway

# Dev (default `make up`): Go gateway only — no embedded Vue. Run the UI with `npm run dev` (Vite on :5173).
FROM debian:bookworm-slim AS dev

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=gateway-build /out/go-mirofish-gateway /app/go-mirofish-gateway
RUN mkdir -p /app/data/projects /app/data/reports /app/data/tasks /app/data/simulations

ENV GATEWAY_BIND_HOST=0.0.0.0
ENV GATEWAY_PORT=3000
ENV PROJECTS_DIR=/app/data/projects
ENV REPORTS_DIR=/app/data/reports
ENV TASKS_DIR=/app/data/tasks
ENV SIMULATIONS_DIR=/app/data/simulations

EXPOSE 3000

ENTRYPOINT ["/app/go-mirofish-gateway"]

# Release: static Vue baked in — use `docker compose -f docker-compose.release.yml up` or `--target release`.
FROM debian:bookworm-slim AS release

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=gateway-build /out/go-mirofish-gateway /app/go-mirofish-gateway
COPY --from=frontend-build /src/frontend/dist /app/frontend-dist
RUN mkdir -p /app/data/projects /app/data/reports /app/data/tasks /app/data/simulations

ENV GATEWAY_BIND_HOST=0.0.0.0
ENV GATEWAY_PORT=3000
ENV FRONTEND_DIST_DIR=/app/frontend-dist
ENV PROJECTS_DIR=/app/data/projects
ENV REPORTS_DIR=/app/data/reports
ENV TASKS_DIR=/app/data/tasks
ENV SIMULATIONS_DIR=/app/data/simulations

EXPOSE 3000

ENTRYPOINT ["/app/go-mirofish-gateway"]
