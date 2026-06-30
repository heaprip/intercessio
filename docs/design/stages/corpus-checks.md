---
tags:
  - устройство
  - веха
---

# corpus-checks — корпус проверяем

Зависит от: [[docs/design/stages/derived-state|derived-state]]  
Компоненты: [[docs/design/components/impact|impact]], [[docs/design/components/linter|linter]], [[docs/design/components/power-graph|power-graph]]

## Состав

`impact` — разница состояний между двумя версиями корпуса. `linter` — проверки
из [[docs/guide/04-what-goes-wrong|каталога провалов]]: `retroactivity`,
`underdetermined-condition`, `delegated-discretion`, `empty-right`,
`duty-without-consequence`, `ladder-without-last-step`, `norm-nobody-applies`,
`impossible-duty`, `competence-gap`, `review-dead-end`, `review-cycle`,
`dangling-succession`, а также цикл в самом отношении вытеснения.

Проверки на графе власти требуют компонента
[[docs/design/components/power-graph|power-graph]].

Эта веха не блокирует игру: она делает её наблюдаемой.

## Демонстрация

Отчёт «правка задела столько-то человек, поимённо».

## Гейт

Не объявлен.
