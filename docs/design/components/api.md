---
tags:
  - устройство
  - компонент
component: api
owns: []
requires:
  - turn
  - report
  - agenda
  - linter
  - censor
  - power-graph
  - scenario
  - entitlement
  - actors
seam: false
---

# api — контракт наружу

## Назначение

Отдать ход и отчёт периода внешнему потребителю.

## Владеет

Ничего из предметной области. Сессия партии — состояние хода, память стопки и
прожитые периоды — живёт здесь, пока хранилища нет (P66).

## Предоставляет

В прототипе — `internal/api`, HTTP с JSON (P65); proto-вариант из
[[docs/decisions/proto-first-external-api|proto-first-external-api]] (`Deprecated`) не
выбран:

| Метод | Путь | Что |
| --- | --- | --- |
| `GET` | `/api/scenarios` | имена сценариев |
| `POST` | `/api/games` | начать партию из сценария; ответ — вид текущего периода |
| `GET` | `/api/games/{id}` | вид текущего периода |
| `GET` | `/api/games/{id}/periods/{n}` | прожитый или текущий период |
| `POST` | `/api/games/{id}/decisions` | решения по карточкам: принять или отклонить |
| `POST` | `/api/games/{id}/advance` | прожить период |
| `GET` | `/api/games/{id}/trace?period=&atom=` | трасса вывода |

Вид периода — стопка и переполнение, находки линтера и цензора, стенд, граф власти,
а у прожитого — отчёт и журнал. Типы вида строятся в `api`; домен их не знает.

## Требует

- `turn` — ход;
- `report` — отчёт периода;
- `agenda`, `linter`, `censor`, `power-graph` — стопка, находки, стенд и граф;
- `scenario` — начальное значение партии;
- `entitlement` — трасса вывода;
- `actors` — заглушки участников.

## Не знает

Доменные события не зависят от транспорта — см.
[[docs/decisions/domain-events-are-transport-agnostic|domain-events-are-transport-agnostic]].

## Исходы и ошибки

Неверное тело — 400; нет партии или периода — 404; карточка не из стопки или
сценарий с ошибками — 422. Имя сценария не может быть путём.

## Шов

Нет.

## Проверяется

`TestAPI_GameThroughHTTP` — гейт вехи [[docs/design/stages/external-view|external-view]].

## Открыто

Ресурсно-ориентированного REST фронту не понадобилось. Открыто: хранение партий, поток
отчётов вместо запросов, участники-модели в сессии.
