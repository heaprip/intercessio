---
tags:
  - adr
  - решение
---

# Ограничить римские названия только presentation layer

Status: Rejected  
Proposed by: не подтверждено — см. [[docs/open-questions|открытые вопросы]]  
Accepted by: — (решение отклонено)  
Зависит от: —  
Заменяет: —

## Context

На раннем этапе рассматривался вариант оставить магистратов, сенат, комиции и
трибунов только в названии продукта и UI, а domain описывать нейтральными
capabilities.

## Decision drivers

- выбранная предметная область теперь сама является римско-вдохновленной
  институциональной симуляцией;
- нормативные документы, статусы и intercessio входят в поведение системы;
- product language и domain language не должны искусственно расходиться.

## Considered options

1. Оставить римский язык только в presentation layer.
2. Использовать его как предметную рамку, уточняя каждый термин собственной
   конституцией и не заявляя историческую точность.
3. Полностью отказаться от римской стилистики.

## Decision

Вариант 1 отклонён. Новое направление описано в решении
[[docs/decisions/roman-inspired-polity-is-the-domain|roman-inspired-polity-is-the-domain]].

## Consequences

### Positive

- история отвергнутой первоначальной границы сохранена;
- римский язык может участвовать в scenarios и normative model.

### Negative / trade-offs

- повышается риск принять историческую ассоциацию за точную competence;
- ubiquitous language требует явных определений вымышленной политии.

## Validation and re-evaluation trigger

Если римская терминология начнет мешать формулировать однозначные invariants,
пересмотреть отдельные имена, не отказываясь автоматически от всей предметной
рамки.

## Links

- Направление, выбранное взамен:
  [[docs/decisions/roman-inspired-polity-is-the-domain|roman-inspired-polity-is-the-domain]],
  само с тех пор заменённое
  [[docs/decisions/stepwise-legislative-game|stepwise-legislative-game]]
