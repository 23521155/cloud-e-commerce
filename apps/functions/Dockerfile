# Runs the Functions host with this app, for the local Docker Compose stack. Azure runs the same
# code on a Function App (Flex Consumption) from the built package instead of this image.
FROM node:22-alpine AS build

WORKDIR /workspace

COPY package.json package-lock.json ./
RUN npm ci

COPY tsconfig.json host.json ./
COPY src ./src
RUN npm run build && npm prune --omit=dev

FROM mcr.microsoft.com/azure-functions/node:4-node22

ENV AzureWebJobsScriptRoot=/home/site/wwwroot \
    AzureFunctionsJobHost__Logging__Console__IsEnabled=true

COPY --from=build /workspace/host.json /workspace/package.json /home/site/wwwroot/
COPY --from=build /workspace/dist /home/site/wwwroot/dist
COPY --from=build /workspace/node_modules /home/site/wwwroot/node_modules
