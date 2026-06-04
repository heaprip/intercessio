# Durable commit и исполнение процессов

Статус: Working design на основе ADR-0006 и ADR-0007  

## Граница атомарности

Application use case не координирует отдельные `Database` и `Broker`. Он
формирует один план фиксации решения. Следующая форма иллюстративна и не является
готовым Go API:

```go
type CommitPlan struct {
	CommitID        string
	StreamID        string
	ExpectedVersion uint64
	Events          []EventEnvelope
	Publications    []Publication
	Effects         []EffectIntent
	Timers          []TimerIntent
}
```

Конкретный consuming interface появляется только вместе с первым use case.
`Publication` — transport-neutral integration intent с устойчивым ID и ordering
key, а не Kafka/YDB/SQS message. Adapter выбирает serialization, topic/queue и
способ возобновления доставки.

Успешный commit не обещает, что HTTP-запрос или внешний authorization write уже
выполнены. Он обещает, что намерение выполнить эффект не потеряется. Обработчик
эффекта допускает повтор, использует idempotency key и сохраняет outcome.

## Реализации без ложной одинаковости

| Среда | Возможная реализация одного контракта |
| --- | --- |
| In-memory | append и deterministic queue под одной критической секцией |
| YDB | row table updates и topic writes в одной serializable transaction |
| AWS | DynamoDB transaction сохраняет state/events и delivery batch; Streams запускает relay в SQS/EventBridge |
| SQL | state/events и delivery rows в одной transaction; poller или CDC доставляет наружу |

Provider-specific capability не протекает в domain. Сильная нативная гарантия
YDB не запрещается ради AWS, а AWS adapter не притворяется физически атомарным с
SQS. Оба должны удовлетворять одинаковому наблюдаемому baseline из ADR-0006.

## Event sourcing

Event sourcing остается выбором конкретного bounded context. Он вероятно полезен
для decision ledger, policy evaluation и долгоживущих процессов, где нужны
temporal reconstruction и audit. Он не назначается по умолчанию для OpenFGA,
справочников, caches и технических projections.

Event sourcing решает dual write только если сохранение события одновременно
создает надежный subscription log. Event table и независимый broker без
transaction, changefeed или relay сохраняют прежний разрыв.

## Workflow kernel

Бизнес-переход проектируется как детерминированное вычисление:

```text
current state + signal -> transition
transition = events + effects + timers
```

Clock, IDs, policy versions и LLM output приходят явными входами. Managed
workflow может доставить signal, retry или timer, но не является каноническим
владельцем business state.

Минимальные runtime capabilities вводятся по необходимости:

- inbox и deduplication входящих signals;
- effect execution с устойчивым ID;
- timers с явно описанной late/duplicate semantics;
- lease/fencing для конкурентного runner;
- reconciliation зависших intents.

## Обязательные contract scenarios

- version conflict не сохраняет частичный plan;
- commit timeout с неизвестным outcome разрешается по `CommitID`;
- падение после commit не теряет publication/effect;
- duplicate delivery не повторяет необратимый результат;
- порядок проверяется только в пределах declared ordering key;
- stale timer или signal не возвращает workflow в прошлую версию;
- reconciliation обнаруживает и возобновляет зависший intent.

## Не принято

- окончательный выбор YDB или AWS;
- универсальный event store для всех contexts;
- broker как source of truth;
- собственный полный workflow engine;
- точная форма Go interfaces до первого вертикального сценария.
