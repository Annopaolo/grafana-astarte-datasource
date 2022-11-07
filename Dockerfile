FROM golang:1.17.8 as gobuilder
WORKDIR /app
ADD . .
RUN go install github.com/magefile/mage@v1.13.0
RUN mage -v

FROM node:14-stretch as jsbuilder
WORKDIR /app
RUN apt-get -qq update
RUN apt-get -qq install netbase build-essential autoconf libffi-dev
COPY --from=gobuilder /app/ .
RUN yarn install
RUN yarn build
ARG GRAFANA_API_KEY
ENV GRAFANA_API_KEY=$GRAFANA_API_KEY
RUN npx grafana-toolkit plugin:sign --signatureType private --rootUrls http://localhost:3000/

FROM grafana/grafana:8.2.6
COPY --from=jsbuilder /app/dist/ /var/lib/grafana/plugins/astarte-datasource/
