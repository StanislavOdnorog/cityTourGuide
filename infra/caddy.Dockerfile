# Build admin static files
FROM node:22-alpine AS admin-builder

WORKDIR /app

COPY admin/package.json admin/package-lock.json ./
RUN npm ci

COPY admin/ .
RUN npm run build

# Caddy with admin static files
FROM caddy:2-alpine

COPY --from=admin-builder /app/dist /srv/admin
