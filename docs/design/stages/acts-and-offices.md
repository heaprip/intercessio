---
tags:
  - устройство
  - веха
---

# acts-and-offices — акты и занятие должностей

Зависит от: [[docs/design/stages/playable-case|playable-case]], [[docs/design/stages/corpus-checks|corpus-checks]]  
Компоненты: [[docs/design/components/case|case]], [[docs/design/components/turn|turn]], [[docs/design/components/linter|linter]], [[docs/design/components/power-graph|power-graph]], [[docs/design/components/censor|censor]]

Веху завёл помощник — решения P59 и дальше в
[[docs/proposals/prototype-decisions|журнале решений]]. Закрывает отложенное решение
«акт и дело» из [[docs/open-questions|открытых вопросов]] гипотезой.

## Состав

1. **Акт должности** — решение без дела: компетенция, кворум, вступление в силу.
2. **Назначение как акт и способ замещения** — ребро «назначает» в графе власти и
   проверка `dangling-succession`.
3. **Узурпация** — назначение неправомочного как событие журнала и находка цензора.

## Демонстрация

Усыновление Клодия прожито ходом: цензор совершает акт, и Клодий становится
допустимым в трибуны без единого ручного факта.

## Гейт

Объявлен помощником: казус о допуске проживается ходом до допуска Клодия, акт вне
компетенции и акт без кворума ничего не меняют, линтер называет непроверяемый акт и
молчит, когда цензоров двое с кворумом.

**Первая часть в прототипе**: `cases.Perform`, `cases.FinalizeAct`, тесты
`TestAdvance_AdoptionIsAnAct`, `TestAdvance_ActNeedsCompetenceAndQuorum`,
`TestLint_UncheckedAct`.
