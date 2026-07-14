FROM node:20-alpine AS deps

WORKDIR /app

# Copy package files for dependency installation
COPY package.json package-lock.json* pnpm-lock.yaml* yarn.lock* ./

# Install production dependencies only
RUN \
  if [ -f yarn.lock ]; then \
    yarn install --frozen-lockfile --production; \
  elif [ -f package-lock.json ]; then \
    npm ci --only=production; \
  elif [ -f pnpm-lock.yaml ]; then \
    corepack enable && pnpm install --frozen-lockfile --prod; \
  else \
    npm install --production; \
  fi

# Builder stage – install dev dependencies and build the Next.js app
FROM node:20-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json* pnpm-lock.yaml* yarn.lock* ./

RUN \
  if [ -f yarn.lock ]; then \
    yarn install --frozen-lockfile; \
  elif [ -f package-lock.json ]; then \
    npm ci; \
  elif [ -f pnpm-lock.yaml ]; then \
    corepack enable && pnpm install --frozen-lockfile; \
  else \
    npm install; \
  fi

COPY . .

RUN npm run build

# Runner stage – minimal runtime image
FROM node:20-alpine AS runner

WORKDIR /app

RUN addgroup --system --gid 1001 nodejs \
  && adduser --system --uid 1001 nextjs

# Copy only what is needed to run the built application
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/package.json ./package.json
COPY --from=builder /app/next.config.ts ./

# Copy production node_modules from the deps stage
COPY --from=deps /app/node_modules ./node_modules

# Ensure correct ownership
RUN chown -R nextjs:nodejs /app

USER nextjs

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"
ENV NODE_ENV=production

CMD ["node_modules/.bin/next", "start"]
