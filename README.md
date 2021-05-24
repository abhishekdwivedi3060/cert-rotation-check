# cert-rotation-check
This repo contains a Go tool to check whether automatic cert rotation is possible or not with given duration and expiry 

# Steps to run:

```shell
go run eg.go -ca-duration=42h -ca-expiry=8h -node-duration=22h -node-expiry=2h -client-duration=9h -client-expiry=2h -min-cert-duration=6h
```

TODO: Make it interactive