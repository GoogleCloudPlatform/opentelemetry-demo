# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0


FROM node:21-alpine AS build

WORKDIR /app

COPY ./test/package*.json ./

RUN npm ci --omit=dev

# -----------------------------------------------------------------------------

FROM node:21-alpine

USER node
WORKDIR /app
ENV NODE_ENV production

COPY --chown=node:node --from=build /app/node_modules/ ./node_modules/
COPY ./test ./
COPY ./pb/demo.proto ./
COPY ./src/productcatalogservice/products/products.json ../src/productcatalogservice/products/products.json

ENTRYPOINT ["npm", "test"]
