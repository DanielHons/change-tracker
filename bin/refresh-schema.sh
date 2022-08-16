#!/usr/bin/env bash

docker-compose -f examples/postgrest/docker-compose.yml  kill -s SIGUSR1 postgrest