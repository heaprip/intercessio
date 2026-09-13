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

Объявлен помощником, ждёт автора. Каждая проверка вехи, которая есть в прототипе,
срабатывает на казусе, ради которого заведена, и молчит на контроле:

- ретроактивная правка `C1` [[docs/design/casus/citizenship|казуса о гражданстве]]
  называет Марка поимённо и не трогает его статус, а та же правка без обратной
  силы не даёт ни замечания, ни затронутых;
- на [[docs/design/casus/office-eligibility|казусе о допуске]] линтер называет
  условие допуска обходимым через `adrogatio` цензора и допуск цензора
  замкнутым, а без компетенции `adrogatio` не называет ни того, ни другого;
- то же на обеих стратегиях разрешения норм.

**Прототип готов, срез вехи сделан, гейт пройден**: `internal/impact`,
`internal/powergraph`, `internal/linter`, тесты `TestLint_*`. В срезе — одиннадцать
провалов каталога (`retroactivity`, `taking-of-vested`,
`underdetermined-condition`, `dead-conclusion`, `competence-gap`,
`review-dead-end`, `review-cycle`, `circumventable-condition`, `closed-eligibility` и частично
`indeterminacy` и `empty-right`) и цикл вытеснения из загрузчика. Сделаны также
`duty-without-consequence` и `ladder-without-last-step`. Не сделаны
`delegated-discretion`, `impossible-duty`, `dangling-succession`; `norm-nobody-applies` — частично, только
должностной путь.

Демонстрация — `TestLint_RetroactiveAmendmentOfC1`, эталон
`internal/linter/testdata/citizenship-c1-retroactive.golden`: правка в периоде 35 с
силой от 28 задела одного человека — Марка, в периодах 30 и 31, когда выносилось
решение о его гражданстве. Граф и замечания казуса о допуске —
`go run ./cmd/lint -now 13 -graph schema.json office-eligibility/scenario.json`.

Что вскрылось:

- **ретроактивная правка бьёт по основанию решения, а не по статусу.** Пересчёт
  отнимает у Марка право на гражданство и полномочие претора его дать ровно в
  тех периодах, когда решение выносилось, а статус держится на вступившем в силу
  решении. Защиту дала форма правил, а не вид нормы;
- **отмена нормы оставляла висящие пары вытеснения** — отмена `A4` давала ошибку
  «правило не существует», пока версия не стала уносить пары вместе с нормой;
- **пересмотр был свёрнут, и тупиком оказывалась почти каждая должность**: тогда
  сдерживание было только intercessio. По решению автора добавлены пересмотр и
  согласие коллег — пока данными и рёбрами графа, без пути обжалования в деле;
- **условие допуска в казусе о допуске действительно обходимо**: цепочка
  `eligible ← holds_right ← status ← adrogatio` находится обходом без единого
  знания о Клодии, и цензор, совершающий акт, замыкает допуск на себя.
