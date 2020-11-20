#!/usr/bin/env bash

docker-compose -f examples/postgrest/env.yml  kill -s SIGUSR1 postgrest