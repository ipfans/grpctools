# grpctools
Additional toolkits for gRPC-go on production.

**⚠️ ATTENTION!!**

* A __work-in-progress__ project, use for your own risks.
* gRPC often changes its API design. So please notes version requirements.

## Naming

### Consul Resolver

The `github.com/ipfans/grpctools/naming/consul` implements new Resolver APIs ([gPRC L9](https://github.com/grpc/proposal/pull/30)) support. It works fine on gRPC-go 1.7.0+. It also can work with new Balancer APIs (e.x. Roundrobin balancer).

## Registery

### Consul Registery

The `github.com/ipfans/grpctools/registery/consul` implements a Registery interface that helps services to register into consul.


## Middleware

### Leaky-bucket Ratelimit

The `github.com/ipfans/grpctools/middleware/ratelimit` implements gRPC Interceptor to rate limit by leaky-bucket rate limit algorith.
