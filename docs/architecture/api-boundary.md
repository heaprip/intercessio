# Граница HTTP и gRPC API

Статус: Working design на основе ADR-0008  

## Канонический contract

Внешний синхронный API описывается `.proto`. Buf управляет schema dependencies,
lint, breaking checks и закрепленной по версиям генерацией. Из одного contract
получаются:

- Go protobuf и grpc types;
- native gRPC server/client stubs;
- gRPC-Gateway HTTP/JSON handlers;
- OpenAPI artifact для документации и внешних clients.

Каждый публичный HTTP endpoint получает явную `google.api.http` annotation.
Сгенерированный REST не считается автоматически хорошим REST: resource paths,
methods, status codes, idempotency и long-running operations проектируются как
публичный продуктовый contract.

## Направление зависимостей

```text
HTTP/JSON or gRPC request
        -> generated transport adapter
        -> application command/query
        -> domain behavior
```

Application и domain не импортируют generated protobuf packages. Mapping
явный, потому что внешний contract и внутренняя модель имеют разные причины для
изменения.

Модули Go-монолита не вызывают друг друга через loopback HTTP/gRPC. Transport
нужен внешним consumers и не моделирует внутреннюю модульную границу.

## Общая policy, разные middleware

Direct gRPC-Gateway registration удобно для serverless unary HTTP, но обходит
gRPC interceptors. Поэтому:

- transport middleware извлекает credentials, headers, tracing и request ID;
- application policy выполняет authorization, validation и idempotency;
- domain повторно защищает собственные invariants;
- mapping errors в HTTP и gRPC является обязанностью adapters.

Нельзя считать наличие gRPC interceptor доказательством, что тот же control
выполнен для REST.

## Serverless deployment

Yandex Cloud Functions получает HTTP event envelope, поэтому не является
предпочтительным host для синхронного API-server. Рабочая гипотеза:

- Serverless Containers обслуживают REST/JSON gateway;
- Cloud Functions исполняют async consumers и triggers;
- native gRPC включается только в deployment с документированным HTTP/2 ingress;
- Yandex Application Load Balancer с gRPC backend рассматривается для VM/K8s
  deployment, если native gRPC станет требованием.

Это deployment hypothesis, а не принятый выбор provider.

## Compatibility gates

- `buf lint` и `buf breaking` для protobuf;
- воспроизводимый `buf generate` с pinned plugins;
- отсутствие незакоммиченного diff generated artifacts;
- breaking/diff check OpenAPI отдельно от protobuf;
- один behavioral suite для HTTP и gRPC adapters;
- явные tests для error mapping, field presence, enums, timestamps, `int64`,
  pagination tokens и idempotency headers.

## Отдельные contracts

Streaming LLM output, progress events, bulk file transfer и webhooks не нужно
насильно пропускать через обычный unary JSON transcoding. Для них допускаются
отдельные SSE/WebSocket, signed URL или message contracts после появления
сценария и отдельного решения.
