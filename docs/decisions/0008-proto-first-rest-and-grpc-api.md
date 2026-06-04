# ADR-0008: Строить внешний API proto-first с REST через gRPC-Gateway

Status: Accepted  
Deciders: author

## Context

Intercessio должен предоставлять обычный HTTP/JSON API и сохранять возможность
native gRPC. Envoy transcoding плохо подходит как обязательная часть serverless
deployment. ConnectRPC упрощает единый RPC-over-HTTP handler, но не дает
ресурсно-ориентированный REST contract и OpenAPI surface без дополнительных
инструментов.

Внешним потребителям важны стандартные URL, HTTP methods, документация и
генерация клиентов. Внутри Go-монолита network transport для межмодульных
вызовов не нужен.

## Decision drivers

- единая protobuf schema для gRPC и HTTP projection;
- явно спроектированный REST API и OpenAPI artifact;
- работа HTTP boundary в serverless deployment без Envoy;
- возможность открыть native gRPC при наличии HTTP/2 ingress;
- контроль backward compatibility через code generation и tests.

## Considered options

1. ConnectRPC как единственный внешний transport.
2. Отдельно проектировать OpenAPI-first HTTP и protobuf gRPC contracts.
3. Protobuf + grpc-go + gRPC-Gateway с явными `google.api.http` annotations.
4. Envoy transcoding как обязательный deployment component.

## Decision

Выбран вариант 3. Buf управляет lint, breaking checks, dependencies и
version-pinned generation. Публичные HTTP bindings задаются явно через
`google.api.http`; `generate_unbound_methods` не используется как замена API
design. Из schema генерируются Go protobuf/grpc types, gRPC-Gateway handlers и
OpenAPI artifact.

REST и native gRPC являются driving adapters над одним application use case.
Protobuf messages не становятся domain types. Модули монолита вызывают
application API напрямую, а не через loopback HTTP/gRPC.

ConnectRPC сейчас не добавляется. Его можно пересмотреть для отдельного
RPC-first или browser streaming interface, если появится такой сценарий.

Для direct in-process gRPC-Gateway registration учитывается, что gRPC
interceptors не выполняются и streaming ограничен. Authorization, idempotency и
критическая validation поэтому принадлежат application policy/decorator, а не
только transport middleware.

## Consequences

### Positive

- обычный REST/JSON доступен curl, OpenAPI tooling и внешним интеграторам;
- одна schema формирует оба transport adapters;
- Envoy не требуется;
- native gRPC можно включить в подходящем deployment.

### Negative / trade-offs

- поддерживаются protobuf, JSON/HTTP и OpenAPI compatibility одновременно;
- gRPC metadata, trailers, streaming и error details переводятся не полностью;
- стабильный gRPC-Gateway generator ориентирован на OpenAPI v2, а v3.1 generator
  пока требует отдельной оценки;
- часть HTTP semantics — status codes, idempotency, ETag, long-running
  operations — проектируется вручную.

## Validation and re-evaluation trigger

Первый публичный API проходит HTTP и gRPC contract tests над одним use case.
Generated OpenAPI проверяется на diff и breaking changes. Пересмотреть выбор,
если API окажется полностью RPC-first, streaming станет доминирующим или
поддержка двух публичных представлений создаст больше расхождений, чем пользы.

## Links

- [gRPC-Gateway introduction](https://grpc-ecosystem.github.io/grpc-gateway/docs/tutorials/introduction/)
- [Buf generation](https://buf.build/docs/generate/)
