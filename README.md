# DEVCHAT Server

Der DevChat-Server nimmt Anfragen über eine REST-Schnittstelle, sowie über Websocket an.

## Konfiguration

```yaml
server:
    addr: # Lokale Serveradresse
    indexFileName: # Dateiname der index.html (Nicht der vollständige Pfad)
    assetsFolder: # Ordner in dem die öffentlich zugänglichen Dateien liegen.

    # Wenn eines dieser Felder leer ist, ist TLS deaktiviert.
    certFile:
    keyFile:
  
database:
    name: # Datenbankname
    addr: # Netzwerkadresse - id:port
    user: # Datenbankbenutzer
    password: # Datenbankpasswort

mailing:
    server: # SMTP-Server
    port: # SMTP-Port (integer)
    password: # Kontopasswort
    user: # Kontobenutzer
    email: # Absender E-Mail Adresse

inmemorydb:
    addr: ""
    password: ""
```

## Wichtigsten Abhänigkeiten

Alle weiteren Abhängigkeiten finden Sie in `go.mod`.

- [jwt-go - JWT Kodierung / Dekodierung](github.com/dgrijalva/jwt-go)
- [go-kit](github.com/go-kit/kit)
- [go-pg - Postgres-Adapter](github.com/go-pg/pg/v9)
- [go-redis - Redis-Adapter](github.com/go-redis/redis)
- [google/uuid - UUID Implementierung](github.com/google/uuid)
- [gorilla/mux - URL Router](github.com/gorilla/mux)
- [gorilla/websocket](github.com/gorilla/websocket)
- [go-diff - Go Port von google-diff-match-patch](github.com/sergi/go-diff)
- [throttled - Rater limiter](github.com/throttled/throttled)
- [yaml.v2 - Unterstützt YAML Parsing](gopkg.in/yaml.v2)
