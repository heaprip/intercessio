---
tags:
  - устройство
  - компонент
component: api
owns: []
requires:
  - turn
  - report
seam: false
---

# api — контракт наружу

## Назначение

Отдать ход и отчёт периода внешнему потребителю.

## Владеет

Не спроектировано.

## Предоставляет

Не спроектировано. Транспорт выбирается заново на вехе
[[docs/design/stages/external-view|external-view]] вместе с первым реальным
контрактом — см.
[[docs/decisions/proto-first-external-api|proto-first-external-api]]
(`Deprecated`).

## Требует

- `turn` — ход;
- `report` — отчёт периода.

## Не знает

Доменные события не зависят от транспорта — см.
[[docs/decisions/domain-events-are-transport-agnostic|domain-events-are-transport-agnostic]].

## Исходы и ошибки

Не спроектировано.

## Шов

Нет.

## Проверяется

Не спроектировано.

## Открыто

Требует ли игровой фронт ресурсно-ориентированного REST вообще — см.
[[docs/open-questions|открытые вопросы]].
