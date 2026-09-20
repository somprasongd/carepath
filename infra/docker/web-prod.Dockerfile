# Production web tier: build the SPA, then serve it as pure static files —
# routing (including /api) lives in the edge proxy, not here. Same source
# layout as web.Dockerfile: floorplans live outside apps/web and are copied
# to the root-relative path the imports expect.
FROM node:22-alpine AS build
WORKDIR /app
COPY apps/web/package*.json ./
RUN npm install
COPY packages/floorplans /packages/floorplans
COPY apps/web .
# Same-origin by design: client.ts defaults to a relative /api path, so the
# bundle needs no API base baked in.
RUN npm run build

FROM nginx:1.27-alpine
COPY infra/docker/web-static.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
