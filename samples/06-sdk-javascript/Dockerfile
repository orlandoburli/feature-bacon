FROM node:20-alpine
WORKDIR /app
COPY sdks/javascript/ /app/sdks/javascript/
COPY samples/06-sdk-javascript/package*.json ./
WORKDIR /app/sdks/javascript
RUN npm ci --ignore-scripts && npm run build
WORKDIR /app
RUN npm install --ignore-scripts
COPY samples/06-sdk-javascript/app.js ./
USER node
EXPOSE 3000
CMD ["node", "app.js"]
