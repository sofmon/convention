# Convention v1

## Configuration

There is no distinction between secrets and configuration values; both must adhere to best practices in secret management as dictated by the hosting environment.

The current convention focuses on containerization, where configuration data is securely retrieved and bound to files.

By default, all configuration files are stored in the `/etc/app` directory.

The following configuration keys/files must be provided by the hosting environment by default:

- `environment`: Specifies the environment's name, with the production environment being named `production`.
- `communication_certificate`: The SSL certificate used for internal HTTPS communication.
- `communication_key`: The SSL certificate key used for internal HTTPS communication.
- `communication_secret`: Secret used to sign and verify authorization tokens employed for internal communication.
- `database`: The necessary configuration for accessing the designated database for the application.

## Communication

All communication uses HTTP protocol with JSON formatted body.

### Secure communication

All communication is through HTTPS using the hosting environment provided `communication_certificate` and `communication_key` as described in the configuration section.

The `communication_certificate` must be registered as trusted certificate on the hosting OS.

### HTTP protocol

Only small subset of HTTP methods and status codes are used for application communication.

HTTP Method defines expectation on request and response body communication as shown in the table below.

|HTTP Method|Request body|Response body|
|---|---|---|
|`GET`|-|JSON|
|`PUT`|JSON|-|
|`DELETE`|-|-|
|`POST`|JSON|JSON|

HTTP Status Code indicates the success of an operation:

|HTTP Status Code|Response body|
|---|---|
|`200 OK`|defined by HTTP Method|
|`409 Conflict`|JSON `error` object|

> Status code `404 Not Found` is not used for application communication. Missing resources are reported with status code `409 Conflict` and `error` object in the response body.

### Error object

All errors are communicate in the HTML body as a JSON object with `code` and `message`:
``` JSON
{
    "code":"...",
    "message":"..."
}
```

### Paths and actions

HTTP `GET` and `PUT` use standard hierarchical path as:
``` HTML
/<parent-resource>/[<parent-resource>/...]<resource>
```

HTTP `POST` path have clear separation of `resource` hierarchical path and `action`:
``` HTML
/<parent-resource>/[<parent-resource>/...]<resource>/@<action>
```

#### Examples

Get all messages for specific user:
```
GET /users/john/messages
```

Put new message for a user
```
PUT /users/john/messages/msg1
```

Send prepared message for a user
```
POST /users/john/messages/msg1/@send
```

### Request ID

Every HTTP request produced by application following this convention should add header `Request-Id` to outgoing HTTP requests.

If the outgoing HTTP request is caused by incoming HTTP request, the same `Request-Id` should be used.

If no `Request-Id` is available, the application would generate new random value for every new outgoing HTTP request.

## Authentication

JWT token is used for internal authentication passed as barer header like:
``` HTML
GET /users/john/messages
Authorization: Bearer <token>
```

All JWT tokens are signed and verified by the `communication_secret` configuration provided by the hosting environment.

The JWT tokens contains set of mandatory claims as shown in the table below.

|Claim|Type|Value description|
|---|---|---|
|`user`|string|Name of the authenticated user|
|`admin`|boolean|Indicating if the authenticated user is an admin|
|`service`|boolean|Indicating if the authenticated user represents a service|

## Database

### Connection and sharding

Database is configured using the `database` configuration. The configuration is a JSON file in the following format:

``` JSON
{
    "engine": "postgres",
    "versions": {
        "v1": {
            "default": {
                "host":"127.0.0.1",
                "port":0,
                "database":"messages_default",
                "username":"some_user",
                "password":"some_password"
            },
            "shards": [
                {
                    "host":"127.0.0.1",
                    "port":0,
                    "database":"messages_shard1",
                    "username":"some_user",
                    "password":"some_password"
                },
                {
                    "host":"127.0.0.1",
                    "port":0,
                    "database":"messages_shard2",
                    "username":"some_user",
                    "password":"some_password"
                }
            ]
        }
    }
}
```

Current convention supports sharding of database from the application layer, where data can be split on multiple schemas/databases or servers.

## Logging

> Current convention targets containerisation where logs are collected form the `stdout`.

All log messages are structured as JSON objects and ends with new line (`'\n'`)

``` JSON
{
    "time": "2023-10-01T01:00:00Z",
    "level": "error",
    "service": "message-v1",
    "user": "josh",
    "message": "✘ unable to connect to database"
}
```

If the log is part of serving an HTTP request, the metadata object contains the HTTP request metadata as shown below.

``` JSON
{
    "time": "2023-10-01T01:00:00Z",
    "level": "error",
    "service": "message-v1",
    "user": "josh",
    "message": "✘ unable to connect to database"
    "metadata": {
        "request_path": "/users/josh/messages",
        "request_method": "GET",
        "request_id": "a9ca2c1a-f993-420d-b851-726dafc35102"
    }
}
```