---
tags:
  - adr
  - решение
---

# Решения

Решение сохраняет причину дорогого или сквозного выбора. Это не протокол
каждого рефакторинга.

Чтобы понять проект, читать это не нужно — для этого есть
[[docs/guide/01-what-it-is|guide]]. Сюда ходят за ответом на вопрос «почему
именно так, а не иначе» и «что придётся переделывать, если передумать».

## Идентификатор

Идентификатором является **слаг имени файла**, не номер: путь
`docs/decisions/<слаг>` в обычном wikilink. Слаг читается прямо в тексте ссылки,
в отличие от номера, за которым надо идти в индекс.

Порядок выражается полями `Зависит от` и `Заменяет`. Хронологию хранит Git; дат
в документах нет.

## Статусы

- `Proposed` — вариант сформулирован, но автор его не принял;
- `Accepted` — текущее основание проектирования;
- `Rejected` — рассмотрен и сознательно не выбран;
- `Superseded` — заменён более новым решением;
- `Deprecated` — больше не рекомендуется, замена ещё не принята.

Решение считается принятым только при `Status: Accepted`. Устное обсуждение,
запись в guide и наличие кода сами по себе статус не меняют.

**`Accepted` проставляет только автор.** Помощник заполняет `Proposed by` и
оставляет `Accepted by` пустым. Принятое решение не редактируют задним числом:
допустимы опечатки, уточнения ссылок и смена статуса, новая логика оформляется
новым решением, которое указывает, что заменяет.

## Индекс

| Решение | Статус | Суть |
| --- | --- | --- |
| [[docs/decisions/stepwise-legislative-game\|stepwise-legislative-game]] | Proposed | Предметная область — пошаговая игра о законотворчестве |
| [[docs/decisions/norms-are-inference-rules\|norms-are-inference-rules]] | Proposed | Условие нормы — правило вывода; вывод трёхзначный, неясность есть исход |
| [[docs/decisions/rights-are-derived-not-stored\|rights-are-derived-not-stored]] | Proposed | Хранятся факты, вычисляются следствия норм |
| [[docs/decisions/documentation-layers\|documentation-layers]] | Proposed | Документация делится по скорости устаревания |
| [[docs/decisions/llm-does-not-grant-authority\|llm-does-not-grant-authority]] | Accepted | Модель не создаёт права, нормы и полномочия |
| [[docs/decisions/modular-monolith\|modular-monolith]] | Accepted | Один deployable с доменными границами внутри |
| [[docs/decisions/domain-events-are-transport-agnostic\|domain-events-are-transport-agnostic]] | Accepted | Доменное событие не зависит от транспорта |
| [[docs/decisions/durable-commit-is-a-semantic-contract\|durable-commit-is-a-semantic-contract]] | Accepted | Durable commit определяется гарантиями, а не механизмом |
| [[docs/decisions/application-owns-workflow-state\|application-owns-workflow-state]] | Accepted | Каноническое состояние процесса принадлежит Intercessio |
| [[docs/decisions/proto-first-external-api\|proto-first-external-api]] | Accepted | Proto-first API с REST через gRPC-Gateway |
| [[docs/decisions/roman-inspired-polity-is-the-domain\|roman-inspired-polity-is-the-domain]] | Accepted | Римско-вдохновлённая полития является предметной областью |
| [[docs/decisions/project-memory-and-documentation-lifecycle\|project-memory-and-documentation-lifecycle]] | Accepted | Разделить оперативную память, документы и историю решений |
| [[docs/decisions/roman-names-in-presentation-only\|roman-names-in-presentation-only]] | Rejected | Отклонено: римские названия только в presentation layer |

## Что здесь требует внимания автора

Четыре решения имеют статус `Proposed` и ждут подтверждения. Два из них
заменяют действующие, поэтому до подтверждения репозиторий находится в
несогласованном состоянии — оно предпочтительнее, чем присвоенная авторитетность:

- `stepwise-legislative-game` заменяет `roman-inspired-polity-is-the-domain`;
- `documentation-layers` заменяет
  `project-memory-and-documentation-lifecycle`.

При принятии каждого: поставить `Accepted`, заполнить `Accepted by`, проставить
заменяемому `Superseded` и строку `Superseded by`, обновить эту таблицу.

**Атрибуция.** Часть принятых решений унаследована с пометкой `Deciders:
author`, но авторство не подтверждено — у них стоит `Proposed by: не
подтверждено`. Подтверждены как авторские: `durable-commit-is-a-semantic-contract`,
`application-owns-workflow-state`, `proto-first-external-api`. Решение
`llm-does-not-grant-authority` предложено помощником и принято автором явно.

**Долг проверки.** У `llm-does-not-grant-authority` критерий гласит «до принятия
построить сценарий, где более сильная модель пытается получить выгодный исход за
пределами компетенции». Сценарий не построен; решение принято. Нужно либо
построить его, либо переформулировать критерий как последующую проверку.
