# ADR-0006: Определять durable commit через гарантии, а не механизм

Status: Accepted  
Deciders: author

**Суть (формулировка автора):** YDB даёт механизм, снимающий dual write
между брокером и базой, но вендор-лока на ней быть не должно.

## Context

Значимая команда должна сохранить авторитетный результат и инициировать
публикации, таймеры или внешние действия. Интерфейсы `Database` и `Broker` не
решают dual write: application service по-прежнему может завершить только одну
из двух операций. YDB умеет включить row tables и topics в одну транзакцию, но
DynamoDB и SQS не дают той же физической операции.

Переносимость нужна между Yandex Cloud и потенциальным AWS, но выравнивание всех
реализаций до одинакового SDK скроет различия, не устраняя их.

## Decision drivers

- отсутствие опубликованного результата без зафиксированного решения;
- гарантированное продолжение работы после успешного commit и падения процесса;
- нативная реализация на YDB без запрета на AWS fallback;
- детерминированные in-memory scenario tests;
- локализация outbox/CDC и provider-specific механики в adapter.

## Considered options

1. Отдельные application ports для database и broker.
2. Универсальный outbox framework как обязательная часть приложения.
3. Один use-case port, описывающий наблюдаемые гарантии durable commit, с
   различными физическими реализациями.
4. Использовать broker как единственный source of truth.

## Decision

Выбран вариант 3. Application layer формирует один commit plan с ожидаемой
версией, доменными фактами и предназначенными для доставки publication, effect
и timer intents. Точное имя и Go-форма появятся из первого сценария, а не из
этого ADR.

Минимальный контракт успешного commit:

1. Авторитетное состояние или события сохранены атомарно относительно plan.
2. При неуспехе публикации и эффекты этого plan не становятся видимыми.
3. После успеха каждый intent будет возобновляемо доставлен как минимум один
   раз, даже если процесс упал сразу после commit.
4. Message/effect ID стабилен, consumer обязан поддерживать idempotency.
5. Порядок гарантируется только для явно заданного ordering key.

Физическая реализация может быть сильнее контракта:

- YDB adapter может записать row table и topic в одной транзакции;
- AWS adapter может атомарно записать state/events и delivery batch через
  `TransactWriteItems`, затем использовать DynamoDB Streams и relay;
- SQL adapter может использовать transactional outbox.

Outbox допустим как внутренняя механика конкретного adapter, но не становится
сквозным application framework. HTTP, OpenFGA, Redis и другие внешние эффекты не
объявляются атомарными с commit: они выполняются из durable intent с retry,
idempotency и при необходимости compensation.

Решение не требует event sourcing для каждого bounded context и не объявляет
broker source of truth.

## Consequences

### Positive

- домен и application не знают топологию provider;
- YDB использует более сильную нативную гарантию без утечки SDK в ядро;
- AWS fallback сохраняет поведение, хотя меняет стоимость и latency;
- один contract test suite можно применить к memory, YDB и AWS adapters.

### Negative / trade-offs

- одинаковый контракт не означает одинаковую эксплуатационную модель;
- AWS adapter все равно содержит relay/outbox-подобный механизм;
- exactly-once processing не обещается, требуется idempotency;
- batching, ordering и transaction limits придется учитывать в дизайне plan.

## Validation and re-evaluation trigger

Первый adapter проходит fault-oriented contract tests: version conflict, crash
после commit, duplicate delivery, retry, stale message и сохранение порядка по
одному key. Пересмотреть контракт, если реальный сценарий требует атомарной
видимости именно во внешнем broker или commit plan не помещается в ограничения
выбранного storage.

## Links

- [YDB transactions](https://ydb.tech/docs/en/concepts/transactions)
- [DynamoDB TransactWriteItems](https://docs.aws.amazon.com/amazondynamodb/latest/APIReference/API_TransactWriteItems.html)
