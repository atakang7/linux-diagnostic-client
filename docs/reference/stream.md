# WebSocket stream

Connect to `/ws` on the HTTP listener. The server checks browser origins against the request host.

| Direction | `type` | `payload` |
| --- | --- | --- |
| Client to server | `view_file` | JSON string with the log file path |
| Server to client | `network` | Array of `NetworkPacket` objects |
| Server to client | `file_update` | One `FileNode` object |
| Server to client | `log` | One `LogEntry` for the selected file |

```json
{"type":"view_file","payload":"/var/log/app.log"}
```

All clients receive network and file update events. A client receives log events only after selecting a matching file. The [runtime reference](runtime.md) records queue and heartbeat limits.
