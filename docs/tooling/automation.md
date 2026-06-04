# Автоматизация проектной памяти и документации

Статус: Working agreement  

## Принято сейчас

### Tool-neutral memory

`AGENTS.md`, `.agents/memory/` и `docs/` читаются любым помощником и живут рядом
с кодом. Это canonical storage. Agent-specific index/cache можно пересоздать.

### Lefthook

В корне лежит `lefthook.yml`. Pre-commit запускает быстрые проверки структуры
документов и staged impact; pre-push дополнительно запускает Go tests/vet, если
в репозитории есть отслеживаемые `.go` файлы. Условие завязано именно на файлы,
а не на `go.mod`: при пустом модуле `go test ./...` возвращает ошибку.

Lefthook закреплен как versioned Go tool: `go.mod` содержит директиву
`tool github.com/evilmartians/lefthook/v2`. Остается задокументировать одну
bootstrap-команду установки хуков. Локальный hook — удобство, CI — обязательная
граница.

### Триггеры двух видов

- hard gate: отсутствующий файл, неверный ADR status/date, ADR без обновленного
  индекса;
- soft signal: изменилось доменное или application code, но документация не
  затронута.

Soft signal намеренно не блокирует commit по умолчанию: file path не доказывает
semantic impact. В CI его можно сделать строгим через
`DOCS_IMPACT_STRICT=1`, когда появится стабильная структура packages.

## Serena: отложить до появления кода

[Serena](https://github.com/oraios/serena) полезна прежде всего как
MCP toolkit для symbol-level поиска, references, diagnostics и рефакторингов
через LSP/JetBrains backend. Она поддерживает Go и имеет собственные memories.

Сейчас подключение преждевременно: символов еще нет, а второй memory store
создаст drift. Вернуться к оценке после первого вертикального среза или когда
обычного `rg`/`gopls` станет недостаточно. Если Serena подключена:

- ее memories считаются cache/индексом, не источником решений;
- onboarding должен указывать на `AGENTS.md` и `docs/`;
- дублирующие проектные facts не коммитятся без необходимости;
- установка выполняется по upstream quick start, а не по случайной marketplace
  конфигурации (это отдельно предупреждают maintainers Serena).

## Кандидаты по мере роста

### Buf и gRPC-Gateway

ADR-0008 выбирает Buf для protobuf lifecycle и gRPC-Gateway для REST projection.
Добавить `buf.yaml`, `buf.lock` и `buf.gen.yaml` вместе с первым реальным внешним
contract. Версии remote plugins закреплять явно. CI должен запускать lint,
breaking check, generation и проверять отсутствие diff.

Generated OpenAPI проверяется отдельно: protobuf wire compatibility не доказывает
совместимость HTTP paths, JSON и public SDK. До появления первой `.proto` schema
не добавлять пустой codegen pipeline.

### Architecture fitness functions

Когда появятся packages, добавить Go-тест или analyzer, запрещающий импорты из
domain в adapters/SDK и нежелательные межконтекстные зависимости. Это надежнее
ручной диаграммы, потому что проверяет фактический dependency graph.

### LikeC4

[LikeC4](https://likec4.dev/) дает architecture-as-code, validation и
генерируемые views. Подключать, когда таблица context map перестанет помещаться в
один экран или появятся минимум три стабильных отношения, которые действительно
нужно визуализировать. До этого DSL и Node toolchain дороже диаграммы.

### AsyncAPI и EventCatalog

[AsyncAPI](https://www.asyncapi.com/docs) описывает внешние message-based APIs;
[EventCatalog](https://www.eventcatalog.dev/) связывает domains, systems,
messages, schemas, flows и ADR и умеет генерировать каталог из спецификаций.
Подключать только после появления внешнего consumer и версионируемого integration
event. Не описывать ими внутренние Go domain events: это снова свяжет ядро с
transport contract.

### Vale и link/schema checks

[Vale](https://docs.vale.sh/) полезен как prose linter при нескольких авторах и
явном style guide. Проверку битых ссылок, JSON Schema/OpenAPI/AsyncAPI и
генерацию reference docs следует добавить вместе с соответствующими файлами, а
не заранее.

## Что не автоматизировать

- принятие ADR;
- перевод `Proposed` в `Accepted`;
- объяснение мотивации задним числом;
- удаление открытого вопроса только потому, что код случайно реализовал один
  вариант;
- обновление `last verified` без фактической проверки;
- создание package/interface из диаграммы.

LLM может подготовить diff и перечислить затронутые документы. Автор принимает
смысл и статус решения.
