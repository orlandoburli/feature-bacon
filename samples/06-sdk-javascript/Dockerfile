FROM node:20-alpine
WORKDIR /build
COPY sdks/javascript/ ./sdks/javascript/
COPY samples/06-sdk-javascript/package*.json ./samples/06-sdk-javascript/
WORKDIR /build/sdks/javascript
RUN npm ci --ignore-scripts && npm run build
WORKDIR /build/samples/06-sdk-javascript
RUN npm install --ignore-scripts --omit=dev
WORKDIR /app
RUN cp -rL /build/samples/06-sdk-javascript/node_modules .
COPY samples/06-sdk-javascript/app.js ./
USER node
EXPOSE 3000
CMD ["node", "app.js"]
