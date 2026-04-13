FROM node:20-alpine
WORKDIR /app
COPY sdks/javascript/ /app/sdks/javascript/
COPY samples/10-backend-feature-flags/package*.json ./
WORKDIR /app/sdks/javascript
RUN npm ci --ignore-scripts && npm run build
WORKDIR /app
RUN npm install --ignore-scripts
COPY samples/10-backend-feature-flags/app.js ./
USER node
EXPOSE 3000
CMD ["node", "app.js"]
