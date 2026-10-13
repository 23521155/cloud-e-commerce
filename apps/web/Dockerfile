FROM node:22-alpine AS build

WORKDIR /app

# postinstall runs `prisma generate`, which needs the schema and config before `npm ci`.
COPY package.json package-lock.json prisma.config.ts ./
COPY prisma ./prisma
RUN npm ci

COPY . .
RUN npm run build

FROM node:22-alpine

WORKDIR /app

ENV NODE_ENV=production
ENV PORT=3000
# The standalone server listens on HOSTNAME; 0.0.0.0 makes it reachable from outside the container.
ENV HOSTNAME=0.0.0.0

# output: "standalone" (next.config.ts) traces only the files the server needs.
# It leaves out public/ and .next/static, so they are copied next to it.
COPY --from=build --chown=node:node /app/.next/standalone ./
COPY --from=build --chown=node:node /app/.next/static ./.next/static
COPY --from=build --chown=node:node /app/public ./public

USER node

EXPOSE 3000

CMD ["node", "server.js"]
