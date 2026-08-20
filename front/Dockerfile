# syntax=docker/dockerfile:1

FROM node:24-alpine AS base
WORKDIR /app
RUN npm i -g pnpm@10

# --- dev: исходники монтируются volume'ом, install выполняется при старте ---
FROM base AS dev
EXPOSE 3000
CMD ["sh", "-c", "pnpm install && pnpm dev --host 0.0.0.0 --port 3000"]

# --- builder ---
FROM base AS builder
COPY package.json pnpm-lock.yaml* ./
RUN pnpm install
COPY . .
RUN pnpm build

# --- prod: Nitro SSR server ---
FROM node:24-alpine AS prod
WORKDIR /app
ENV NODE_ENV=production HOST=0.0.0.0 PORT=3000
COPY --from=builder /app/.output ./.output
USER node
EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]
