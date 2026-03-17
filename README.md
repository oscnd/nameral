# Nameral

Name server with dynamic zone resolving

## Development

1. Create certificate for server

    ```bash
    openssl req -x509 -newkey ed25519 -keyout .local/server.key -out .local/server.crt -days 3650 -nodes -subj "/CN=nameral" -addext "subjectAltName=IP:127.0.0.1,DNS:localhost"
    ```
   
2. Setup backend configuration in `.local/config.yml`

    ```yaml
    appName: nameral
    webListen:
      - tcp
      - ":8080"
    protoListen:
      - tcp
      - ":50051"
    dnsListen: ":53"
    telemetryUrl: "http://localhost:4318"
    redisAddress: "localhost:6379"
    redisDatabase: 0
    serverCertificateFile: ".local/server.crt"
    serverPrivateKeyFile: ".local/server.key"
    clients:
      - name: client1
        token: secret01
        allowedZones:
          - "."
    ```

3. Setup agent configuration in `.local/config.agent.yml`

    ```yaml
    address: "localhost:50051"
    secret: "secret01"
    zones:
      - "lan.bsthun.in"
    upstream: "10.2.1.1:53"
    certificateFile: ".local/server.crt"
    ```

4. Run server

    ```bash
    BACKEND_CONFIG_FILE=.local/config.yml go run ./command/backend
    ```
   
5. Run agent

    ```bash
   BACKEND_CONFIG_FILE=.local/config.agent.yml go run ./command/agent
    ```