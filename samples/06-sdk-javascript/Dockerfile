FROM node:20-alpine
WORKDIR /app
COPY sdks/javascript/ /app/sdks/javascript/
COPY samples/06-sdk-javascript/package*.json ./
RUN cd /app/sdks/javascript && npm ci && npm run build && cd /app && npm install
COPY samples/06-sdk-javascript/app.js ./
EXPOSE 3000
CMD ["node", "app.js"]
