#!/usr/bin/env bash

docker-compose -f examples/postgrest/env.yml down && docker-compose -f examples/postgrest/env.yml up -d