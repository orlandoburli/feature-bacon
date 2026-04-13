FROM node:20-alpine
WORKDIR /build
COPY sdks/javascript/ ./sdks/javascript/
COPY samples/09-ui-feature-flags/package*.json ./samples/09-ui-feature-flags/
WORKDIR /build/sdks/javascript
RUN npm ci --ignore-scripts && npm run build
WORKDIR /build/samples/09-ui-feature-flags
RUN npm install --ignore-scripts --omit=dev
WORKDIR /app
RUN cp -rL /build/samples/09-ui-feature-flags/node_modules .
COPY samples/09-ui-feature-flags/app.js ./
USER node
EXPOSE 3000
CMD ["node", "app.js"]
