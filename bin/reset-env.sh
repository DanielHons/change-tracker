#!/usr/bin/env bash

docker-compose -f examples/postgrest/docker-compose.yml down && docker-compose -f examples/postgrest/docker-compose.yml up -d