# Serverless storage, authorization, workflow и API transport

Question: Какие provider capabilities и Go libraries позволяют сохранить
архитектурные гарантии Intercessio без ложной переносимости?

## Findings

### OpenFGA допускает собственный YDB datastore

OpenFGA — не только client SDK. Go server экспортирует
`storage.OpenFGADatastore`, а `server.WithDatastore` принимает его реализацию.
Собственный YDB adapter можно подключить без fork ядра, но потребуется свой
composition root/executable: стандартная конфигурация знает только memory,
PostgreSQL, MySQL и SQLite.

Работа adapter включает не только tuple CRUD: reverse reads, pagination,
authorization models, assertions, stores, changelog, transactions и iterator
semantics. OpenFGA публикует server test/benchmark packages, которые следует
использовать вместе с fault и load tests.

Для такой реализации предпочтительны native YDB row tables и схема от access
patterns OpenFGA. Наличие custom datastore не делает доменный commit и вызов
OpenFGA API одной транзакцией. Relationship tuples вероятно будут projection
доменных фактов; security-critical revoke требует отдельного consistency design.

### YDB и AWS реализуют один contract разными механизмами

YDB поддерживает транзакции с row tables и topics: topic write становится
видимым при commit. Это дает нативную реализацию durable state + publication.

DynamoDB `TransactWriteItems` атомарно изменяет items, но не включает SQS. Для
аналогичного наблюдаемого результата adapter сохраняет delivery batch вместе со
state/events, после чего DynamoDB Streams запускает relay. Это outbox-подобная
механика внутри AWS adapter, а не обязательный application framework.

DynamoDB Streams гарантирует порядок изменений одного item, но не следует
неявно считать это порядком нескольких event items одного transaction. Ordering
key и batch representation требуют отдельного дизайна и tests.

### Event sourcing не устраняет любую dual write проблему

Описанный Microservices.io вариант предполагает event store, который сам
является subscription log/broker. Event table и независимый broker без
transaction, changefeed или relay сохраняют окно потери.

Event sourcing полезен там, где события действительно являются канонической
историей и нужны replay/temporal audit. Решение применять его ко всем contexts
Intercessio не принято.

### Managed workflow нельзя портировать единым широким interface

Yandex Workflows использует YaWL и JSON state, AWS Step Functions — ASL и
собственное execution state. Общий `WorkflowEngine` либо протечет vendor
semantics, либо станет собственным DSL/compiler.

Переносимый слой — детерминированный business transition и минимальные
capabilities signals/timers/effects/leases. Managed engine может быть runner,
если каноническое business state остается в Intercessio.

### gRPC-Gateway выбран для публичной HTTP projection

ConnectRPC предоставляет один `net/http` handler для Connect, gRPC и gRPC-Web и
остается сильным вариантом для RPC-first controlled clients. Он не заменяет
осознанно спроектированный resource-oriented REST/OpenAPI contract.

gRPC-Gateway генерирует HTTP/JSON handlers по `google.api.http`. Direct
in-process registration подходит unary serverless API, но не выполняет gRPC
interceptors и ограничивает streaming. Поэтому security controls располагаются
в application layer, а не только в transport middleware.

Стабильный OpenAPI generator gRPC-Gateway ориентирован на v2; генератор 3.1
имеет alpha status. Yandex API Gateway заявляет OpenAPI 3.0, поэтому конвертация
и compatibility check остаются отдельной задачей.

### Yandex serverless разделяет sync API и functions

Cloud Functions получает HTTP request как JSON event envelope, а Serverless
Containers вызываются по HTTPS. Документированного user gRPC ingress для
Serverless Containers не найдено; HTTP/2 + gRPC существует как feature request.
Application Load Balancer поддерживает gRPC/HTTP2 для сетевых backend targets.

Рабочее следствие: Serverless Containers для sync REST gateway, Functions для
async consumers. Native gRPC требует подходящего ingress/deployment и не
считается доступным автоматически.

## Sources

- [OpenFGA storage package](https://pkg.go.dev/github.com/openfga/openfga/pkg/storage)
- [OpenFGA server WithDatastore](https://pkg.go.dev/github.com/openfga/openfga/pkg/server)
- [OpenFGA configuration](https://openfga.dev/docs/getting-started/setup-openfga/configuration)
- [OpenFGA server tests](https://pkg.go.dev/github.com/openfga/openfga/pkg/server/test)
- [YDB transactions](https://ydb.tech/docs/en/concepts/transactions)
- [YDB topics](https://ydb.tech/docs/en/concepts/topic)
- [DynamoDB TransactWriteItems](https://docs.aws.amazon.com/amazondynamodb/latest/APIReference/API_TransactWriteItems.html)
- [DynamoDB Streams](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Streams.html)
- [Microservices.io Event sourcing](https://microservices.io/patterns/data/event-sourcing.html)
- [Yandex Workflows](https://yandex.cloud/en/docs/serverless-integrations/concepts/workflows/workflow)
- [AWS Step Functions state machines](https://docs.aws.amazon.com/step-functions/latest/dg/concepts-statemachines.html)
- [gRPC-Gateway](https://grpc-ecosystem.github.io/grpc-gateway/docs/tutorials/introduction/)
- [gRPC-Gateway direct registration API](https://pkg.go.dev/github.com/grpc-ecosystem/grpc-gateway/v2/examples/internal/helloworld)
- [gRPC-Gateway OpenAPI 3.1 status](https://grpc-ecosystem.github.io/grpc-gateway/docs/mapping/openapi_v3/)
- [ConnectRPC](https://connectrpc.com/)
- [Buf generation](https://buf.build/docs/generate/)
- [Yandex Cloud Functions invocation](https://yandex.cloud/ru/docs/functions/concepts/function-invoke)
- [Yandex Serverless Containers invocation](https://yandex.cloud/en/docs/serverless-containers/concepts/invoke)
- [Yandex ALB backend groups](https://yandex.cloud/ru/docs/application-load-balancer/concepts/backend-group)

## Applicability to Intercessio

Исследование является основанием ADR-0006, ADR-0007 и ADR-0008. Оно не принимает
cloud provider, event sourcing scope, OpenFGA deployment или форму первого
workflow.

## Unknowns and expiry trigger

Повторно проверить перед implementation spike:

- версии и ограничения YDB table/topic transaction SDK для Go;
- актуальный `OpenFGADatastore` contract выбранной версии OpenFGA;
- наличие native gRPC ingress у Yandex Serverless Containers;
- стабильность gRPC-Gateway OpenAPI 3 generator;
- реальные transaction size, ordering и latency limits выбранного provider.
