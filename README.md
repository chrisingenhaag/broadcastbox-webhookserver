# BroadcastBox Webhook Server

A minimal Go server for authenticating webhook requests using bearer tokens mapped to stream keys via environment variables.

**This server is built to be used together with [github.com/glimesh/broadcast-box](https://github.com/glimesh/broadcast-box).**


This server allows asynchronous definition of a secret stream key and a string representing the name of the stream for your watchers. This enables private streams without the need to expose the key to allow streaming (WHIP).

**Example:**

Suppose you want to allow a streamer to ingest using a secret key, but let viewers watch using a public stream name:

```
WEBHOOK_ENABLED_STREAMKEYS="supersecretkey:myprivatestream"
```

- The streamer uses `supersecretkey` to start streaming (WHIP connect).
- Viewers/watchers use `myprivatestream` to watch (WHEP connect), without ever seeing the secret key.

## Features
- Accepts POST requests with a JSON payload
- Authenticates using bearer tokens
- Returns the mapped stream key if authentication succeeds
- Logs invalid attempts

## Environment Variable

Set `WEBHOOK_ENABLED_STREAMKEYS` to a comma-separated list of `bearerToken:streamKey` pairs. Example:

```
WEBHOOK_ENABLED_STREAMKEYS="token1:streamkey1,token2:streamkey2"
```

## Usage

### Run Locally

1. Clone the repository:
   ```sh
   git clone https://github.com/chrisingenhaag/broadcastbox-webhookserver.git
   cd broadcastbox-webhookserver
   ```
2. Set the environment variable:
   ```sh
   export WEBHOOK_ENABLED_STREAMKEYS="token1:streamkey1,token2:streamkey2"
   ```
3. Start the server:
   ```sh
   go run main.go
   ```
   The server listens on port 8000 by default.

### Example Request

```
curl -X POST http://localhost:8000/ \
  -H "Content-Type: application/json" \
  -d '{
    "action": "test",
    "ip": "127.0.0.1",
    "bearerToken": "token1",
    "queryParams": {},
    "userAgent": "curl"
  }'
```

#### Successful Response
```
{"streamKey":"streamkey1"}
```

#### Invalid Token Response
```
{"streamKey":""}
```

## Docker Deployment

1. Build the Docker image:
   ```sh
   docker build -t broadcastbox-webhookserver .
   ```
2. Run the container:
   ```sh
   docker run -p 8000:8000 -e WEBHOOK_ENABLED_STREAMKEYS="token1:streamkey1,token2:streamkey2" broadcastbox-webhookserver
   ```

## Testing

Run tests with:
```sh
go test
```

## License

Apache License 2.0
