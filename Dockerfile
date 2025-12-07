# Polyglot LLM Gateway - TypeScript/Node.js
# Multi-stage build for minimal image size

# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Install pnpm
RUN corepack enable && corepack prepare pnpm@latest --activate

# Copy package files first for better caching
COPY ts/package.json ts/pnpm-lock.yaml ts/pnpm-workspace.yaml ./
COPY ts/packages/gateway-core/package.json ./packages/gateway-core/
COPY ts/packages/gateway-adapter-node/package.json ./packages/gateway-adapter-node/
COPY ts/apps/gateway-node/package.json ./apps/gateway-node/

# Install dependencies
RUN pnpm install --frozen-lockfile

# Copy source code
COPY ts/packages ./packages
COPY ts/apps/gateway-node ./apps/gateway-node

# Build all packages
RUN pnpm -r build

# Production stage
FROM node:20-alpine AS runner

WORKDIR /app

# Install pnpm for production
RUN corepack enable && corepack prepare pnpm@latest --activate

# Copy package files
COPY ts/package.json ts/pnpm-lock.yaml ts/pnpm-workspace.yaml ./
COPY ts/packages/gateway-core/package.json ./packages/gateway-core/
COPY ts/packages/gateway-adapter-node/package.json ./packages/gateway-adapter-node/
COPY ts/apps/gateway-node/package.json ./apps/gateway-node/

# Install production dependencies only
RUN pnpm install --frozen-lockfile --prod

# Copy built files from builder
COPY --from=builder /app/packages/gateway-core/dist ./packages/gateway-core/dist
COPY --from=builder /app/packages/gateway-adapter-node/dist ./packages/gateway-adapter-node/dist
COPY --from=builder /app/apps/gateway-node/dist ./apps/gateway-node/dist

# Copy config example
COPY config.example.yaml ./config.yaml

# Set environment
ENV NODE_ENV=production
ENV PORT=8080

EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the gateway
WORKDIR /app/apps/gateway-node
CMD ["node", "dist/index.js"]
