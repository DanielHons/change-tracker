# change-tracker

## Run locally
Set the following environment variables

Run the environment
```
docker-compose -f env/dev/docker-compose.yml up -d
```

| Name | Value |
|------|-------|
|DATABASE_CONNECTION_STRING| postgres://migration_user:thisWillBeDifferent@localhost:5432/app_db?sslmode=disable|
|MIGRATION_FILES| Absolute path of the folder sql/schema|
|NOTIFIER_API_TOKEN| Your notification token. Keep it secret and only send it via HTTPS |
|TOKEN_HEADER_IN| Name of the header to look for the API token |
|JWKS_URL| Where to find the JWK set used to authenticate users performing the diff  |


## Notifications
Notifications are received via the proxy and directly inserted into the database, using the same connection as to migrate the schema.

A valid notifiication looks like this
```
## Notify
curl -X "POST" "http://localhost:4711/notify/single" \
     -H 'Authorization: <Your static API token>' \
     -H 'Content-Type: application/json; charset=utf-8' \
     -d $'{
  "component": "my-fancy-service",
  "version": "v2.3.4",
  "sha": "ed73927",
  "stage": "test-environment-a"
}'

```

It is also possible to notify multiple stages (for the same commit) at once:
```
## Notify
curl -X "POST" "http://localhost:4711/notify/multi" \
     -H 'Authorization: <Your static API token>' \
     -H 'Content-Type: application/json; charset=utf-8' \
     -d $'{
  "component": "my-fancy-service",
  "version": "v2.3.4",
  "sha": "ed73927",
  "stages": ["stage1","stage2"]
}'

```

It is totally up to you what a `stage`is 

For a specific stage and component, an alias for the stage can be definied. The diff between environments is based on the alias if present and the plain stage otherwise.