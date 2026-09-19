FROM node:22-alpine
WORKDIR /app
COPY apps/web/package*.json ./
RUN npm install
# Floor-plan SVG assets live outside apps/web (packages/floorplans); the
# navigate screen imports them via ../../../../../packages/floorplans —
# copy them to the same root-relative path so the imports resolve here too.
COPY packages/floorplans /packages/floorplans
COPY apps/web .
EXPOSE 5173
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]
