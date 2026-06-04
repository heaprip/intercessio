# Architecture Decision Records

ADR сохраняет причину дорогого или сквозного решения. Он не является протоколом
каждого рефакторинга.

## Статусы

- `Proposed` — вариант сформулирован, но автор его не принял;
- `Accepted` — текущее основание проектирования;
- `Rejected` — рассмотрен и сознательно не выбран;
- `Superseded` — заменен более новым ADR;
- `Deprecated` — больше не рекомендуется, замена еще не принята.

Решение считается принятым только при `Status: Accepted`. Устное обсуждение,
memory-файл и наличие кода сами по себе статус не меняют.

Статус `Accepted` проставляет только автор. Помощник заполняет `Proposed by` и
оставляет `Accepted by` пустым; черновик до принятия живет в `docs/proposals/`.

## Индекс

| ADR | Статус | Решение |
| --- | --- | --- |
| [0001](0001-modular-monolith.md) | Accepted | Один deployable с доменными границами внутри |
| [0002](0002-domain-events-are-transport-agnostic.md) | Accepted | Доменное событие не зависит от транспорта |
| [0003](0003-llm-does-not-grant-authority.md) | Accepted | LLM не выдает права и не исполняет необратимые действия |
| [0004](0004-roman-names-are-product-metaphor.md) | Rejected | Отклонено: ограничить римские названия только presentation layer |
| [0005](0005-project-memory-and-documentation-lifecycle.md) | Accepted | Разделить оперативную память, долговечные документы и историю решений |
| [0006](0006-durable-commit-is-a-semantic-contract.md) | Accepted | Определять durable commit через наблюдаемые гарантии |
| [0007](0007-application-owns-workflow-state.md) | Accepted | Хранить каноническое workflow state в Intercessio |
| [0008](0008-proto-first-rest-and-grpc-api.md) | Accepted | Proto-first API с REST через gRPC-Gateway |
| [0009](0009-roman-inspired-polity-is-the-domain.md) | Accepted | Римско-вдохновленная полития является предметной областью |

ADR не датируются. Порядок решений выражается полями `Зависит от` и
`Заменяет`, а не хронологией: даты в этом репозитории не несут информации,
а хронологию хранит Git.

При добавлении ADR обновить эту таблицу и запустить `scripts/check-docs.sh`.
