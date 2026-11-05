# Security design

DeckDrop has no accounts and no server in the middle. Devices trust each other by pinning certificates, set up once with a short pairing code.

## Identity and encryption

Each device creates a self-signed certificate the first time it runs (`identity.crt` and `identity.key` in the config folder; the key is readable only by you). Its SHA-256 fingerprint identifies the device and is advertised in the mDNS `fp=` record. All transfers are encrypted with TLS 1.3.

## Pairing

The receiver shows a one-time six digit code. The code is never sent: the other device sends an HMAC of both certificate fingerprints keyed with the code, so someone watching the network, or sitting in the middle with their own certificate, cannot reuse it. The code expires after two minutes, works once, and is discarded after five wrong attempts.

Each side stores the other's certificate fingerprint in `paired.json` in the config folder, and the sender pins it. The server requests a client certificate and accepts requests only from paired fingerprints, so the certificate is the credential and there are no tokens.

## Offers

A paired device must offer its files first. The receiver's user accepts or declines, and an unanswered offer is declined after a minute. Nothing is written until an offer is accepted, and only the offered files, up to their offered sizes, are accepted. Grants are held in memory, are removed when a file completes, and expire after a day unused, so resuming does not ask again.

## Visibility

By default only paired devices are listed. Pairing mode lists nearby unpaired devices for two minutes. The **Hide this device** setting stops mDNS announcing entirely.

## Trust model and limits

- There is no certificate authority. A sender checks that a device's certificate matches the fingerprint it has pinned for it.
- Connections to a paired device are authenticated: the sender refuses a certificate that is not the pinned one. The fingerprint advertised over mDNS is only a hint for finding the pin; anyone can announce anything, but cannot present the paired device's certificate.
- Pairing is only as strong as the code: an attacker has five guesses at a one in a million code per pairing attempt, and the person pairing should check the code appears on the device they expect.
- mDNS is multicast on the local network. Hiding reduces what is listed and who can send, but does not hide a device from someone sniffing the network.
- Deleting the identity files gives the device a new identity, which devices that pinned the old one will reject.

## HTTP API

Files are sent in chunks so interrupted transfers can resume. The server speaks HTTPS only (TLS 1.3).

| Endpoint                   | Description                                                                                                                 |
| -------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| `GET /health`              | Returns 200 when the backend is running                                                                                     |
| `POST /pair`               | Pairs a device that proves it knows the one-time code                                                                       |
| `POST /offer`              | Asks the receiver to accept a list of files (`{"files":[{"name","size"}]}`). Waits for the user: 204 accepted, 403 declined |
| `POST /upload`             | Writes the request body at a byte offset. Headers: `X-File-Name`, `X-File-Offset`. Returns 204                              |
| `GET /status?filename=...` | Returns the bytes received so far in the `X-File-Offset` header, to resume from                                             |

`/offer`, `/upload` and `/status` return 403 unless the caller presented the certificate of a paired device. `/upload` and `/status` also return 428 for a file that was not offered and accepted, and an upload cannot go past the size that was offered. Uploads over the maximum chunk size (64 MiB) are rejected with 413. Invalid file names are rejected with 400.

Only `/health` and `/pair` can be called without pairing, for example `curl -k https://localhost:8080/health` (`-k` because the certificate is self-signed).
