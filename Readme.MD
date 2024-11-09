# dev-proxy
An implementation of monzo's [devproxy](https://github.com/gwuah/dev-proxy) for my use.

# context
You can read the monzo blog for context on why such a tool is useful.  
Unfortunately, I'm back to dealing with microservices and I badly need a tool like this.

# usage
- Create a folder called keys
- Generate some keys & certs inside that folder (using commands below)
```
openssl genrsa -out ca.key 2048
openssl req -new -x509 -days 365 -key ca.key -out ca.crt
```
- To use the proxy, run `make build` & `make run`
- Configure your http client to use `localhost:7777` as a proxy.
- To view request payloads, use the viewer running on `localhost:6666`


# design
This is not complex software. It has 3 components.
- an http proxy
- an sqlite database
- a webpage to view traffic

# features
- [] proxying
- [] view traffic
- [] service virtualization
