FROM node:20-alpine
WORKDIR /build
COPY sdks/javascript/ ./sdks/javascript/
COPY samples/10-backend-feature-flags/package*.json ./samples/10-backend-feature-flags/
WORKDIR /build/sdks/javascript
RUN npm ci --ignore-scripts && npm run build
WORKDIR /build/samples/10-backend-feature-flags
RUN npm install --ignore-scripts --omit=dev
WORKDIR /app
RUN cp -rL /build/samples/10-backend-feature-flags/node_modules .
COPY samples/10-backend-feature-flags/app.js ./
USER node
EXPOSE 3000
CMD ["node", "app.js"]
