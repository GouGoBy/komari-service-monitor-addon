FROM node:22-bookworm-slim AS build
WORKDIR /src
RUN npm install --global pnpm@11.25.0
COPY komari-web/package.json komari-web/pnpm-lock.yaml komari-web/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile --ignore-scripts
COPY komari-web/ ./
RUN ./node_modules/.bin/tsc -b && ./node_modules/.bin/vite build

FROM nginx:stable-alpine
COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist/ /usr/share/nginx/html/
COPY --from=build /src/komari-theme.json /usr/share/nginx/html/themes/default/komari-theme.json
EXPOSE 80
