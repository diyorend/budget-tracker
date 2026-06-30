# Bun-based build, matching the project's actual package manager (bun.lock,
# not package-lock.json — the original Dockerfile used `npm ci`, which
# either fails outright with no package-lock.json present or silently
# ignores the bun.lock and resolves different dependency versions).
FROM oven/bun:1-alpine AS builder
WORKDIR /app

COPY package.json bun.lock ./
RUN bun install --frozen-lockfile

COPY . .

# Vite only inlines import.meta.env.VITE_* values at BUILD time, not
# runtime — docker-compose passes them as `build.args`, but without these
# ARG/ENV lines the build stage never sees them and the frontend silently
# falls back to its hardcoded localhost defaults in every environment.
ARG VITE_API_URL
ARG VITE_WS_URL
ENV VITE_API_URL=$VITE_API_URL
ENV VITE_WS_URL=$VITE_WS_URL

RUN bun run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx-frontend.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
