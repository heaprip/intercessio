# ADR-0007: Intercessio владеет каноническим состоянием workflow

Status: Accepted  
Deciders: author

**Суть (формулировка автора):** решение планируется serverless, поэтому,
вероятно, понадобится самописный workflow-движок. Сейчас достаточно продумать
порты так, чтобы локально работали обычные решения (Postgres, Kafka/NATS,
Temporal, Redis).

## Context

Долгоживущие решения требуют signals, timers, retries и внешних эффектов.
Yandex Workflows и AWS Step Functions могут исполнять такие процессы, но
хранят собственное состояние и используют разные vendor DSL. Если business
state и переходы существуют только внутри managed engine, перенос незавершенных
процессов становится отдельной миграцией.

Одновременно преждевременно строить собственный полный аналог Temporal или
Step Functions до появления первого workflow.

## Decision drivers

- provider fallback без переписывания бизнес-переходов;
- нативные Go-тесты с контролируемыми clock и queue;
- воспроизводимость решения и аудита;
- возможность использовать managed retries/timers;
- минимизация собственного infrastructure framework на старте.

## Considered options

1. Хранить workflow definition и state только в managed engine.
2. Создать собственный универсальный durable execution engine немедленно.
3. Владеть бизнес-состоянием и переходами в Intercessio, используя managed
   engine как runner/scheduler там, где это полезно.

## Decision

Выбран вариант 3. Бизнес-переход выражается детерминированной функцией по смыслу
`Advance(state, signal) -> transition`. Transition содержит факты, effect и
timer intents и фиксируется через контракт ADR-0006.

Managed workflow не является system of record. Он может будить execution,
повторять task, ждать callback и планировать время, но канонический status,
основание решения и ожидаемая версия принадлежат Intercessio.

Собственный runtime ограничивается фактически понадобившимися capabilities:
inbox/deduplication, effects, timers и leases. Общий workflow DSL, compiler и
универсальный engine заранее не создаются.

## Consequences

### Positive

- переходы симулируются обычным `go test`;
- смена runner не меняет доменную модель;
- падение runner не уничтожает бизнес-состояние;
- audit и workflow используют одну модель commit.

### Negative / trade-offs

- часть durable execution обязанностей остается приложению;
- преимущества vendor-native state machine используются не полностью;
- придется определять fencing, lease expiry, deduplication и timer semantics;
- миграция active executions все равно потребует operational plan.

## Validation and re-evaluation trigger

Проверить на первом процессе, который действительно переживает несколько
invocations. Пересмотреть, если собственные timers/leases/effects становятся
сложнее управляемого engine или если выбранный provider дает переносимый способ
экспортировать и продолжать in-flight state без потери аудита.
