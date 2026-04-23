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

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/go-mirofish-gateway ./cmd/mirofish-gateway

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=gateway-build /out/go-mirofish-gateway /app/go-mirofish-gateway
COPY --from=frontend-build /src/frontend/dist /app/frontend-dist

ENV GATEWAY_BIND_HOST=0.0.0.0
ENV GATEWAY_PORT=3000
ENV FRONTEND_DIST_DIR=/app/frontend-dist
ENV BACKEND_URL=http://backend:5001

EXPOSE 3000

ENTRYPOINT ["/app/go-mirofish-gateway"]
