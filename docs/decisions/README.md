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
запись в guide и наличие кода сами по себе статус не меняют. На `Proposed`
проектирование не опирается.

**`Accepted` — акт автора.** Автор ставит его сам либо прямо указывает помощнику
проставить его у названных решений; тогда `Accepted by` читается как `author
(проставлено помощником по прямому указанию автора)`. Во всех остальных случаях
помощник заполняет `Proposed by` и оставляет `Accepted by` пустым.

Принятое решение не редактируют задним числом: допустимы опечатки, уточнения
ссылок и смена статуса, новая логика оформляется новым решением, которое
указывает, что заменяет.

## Индекс

| Решение | Статус | Суть |
| --- | --- | --- |
| [[docs/decisions/stepwise-legislative-game\|stepwise-legislative-game]] | Accepted | Предметная область — пошаговая игра о законотворчестве |
| [[docs/decisions/player-works-a-stack-of-proposals\|player-works-a-stack-of-proposals]] | Accepted | Период проживается пачкой; игрок разбирает стопку предложений правки закона или пишет норму сам, но дел не решает |
| [[docs/decisions/norms-are-inference-rules\|norms-are-inference-rules]] | Accepted | Условие нормы — опровержимое правило; исключение вытесняет, а не отрицает |
| [[docs/decisions/document-hierarchy-fills-the-defeat-relation\|document-hierarchy-fills-the-defeat-relation]] | Accepted | Иерархия документов — источник вытеснения; объявленная пара сильнее |
| [[docs/decisions/rights-are-derived-not-stored\|rights-are-derived-not-stored]] | Accepted | Хранятся факты, вычисляются следствия норм |
| [[docs/decisions/duties-have-a-lifecycle\|duties-have-a-lifecycle]] | Accepted | Обязанность имеет носителя, срок, вид и выводимое нарушение |
| [[docs/decisions/enforcement-is-finite\|enforcement-is-finite]] | Accepted | Полития не всеведуща: презумпция исполнения, проверка — ресурс |
| [[docs/decisions/practice-becomes-norm-only-by-enactment\|practice-becomes-norm-only-by-enactment]] | Accepted | Практика извлекается как находка; нормой её делает auctor |
| [[docs/decisions/extension-points-require-a-casus\|extension-points-require-a-casus]] | Accepted | Точка расширения оправдана двумя вариантами из казуса |
| [[docs/decisions/documentation-layers\|documentation-layers]] | Accepted | Документация делится по скорости устаревания |
| [[docs/decisions/design-layer-has-a-spine-and-component-cards\|design-layer-has-a-spine-and-component-cards]] | Accepted | Архитектура читается подряд, компоненты — в проверяемых карточках, понятия — по областям; номер только там, где порядок и есть смысл |
| [[docs/decisions/llm-does-not-grant-authority\|llm-does-not-grant-authority]] | Accepted | Модель не создаёт права, нормы и полномочия |
| [[docs/decisions/modular-monolith\|modular-monolith]] | Accepted | Один deployable с доменными границами внутри |
| [[docs/decisions/domain-events-are-transport-agnostic\|domain-events-are-transport-agnostic]] | Accepted | Доменное событие не зависит от транспорта |
| [[docs/decisions/durable-commit-is-a-semantic-contract\|durable-commit-is-a-semantic-contract]] | Accepted | Durable commit определяется гарантиями, а не механизмом |
| [[docs/decisions/application-owns-workflow-state\|application-owns-workflow-state]] | Accepted | Каноническое состояние процесса принадлежит Intercessio |
| [[docs/decisions/proto-first-external-api\|proto-first-external-api]] | Deprecated | Proto-first API с REST через gRPC-Gateway; выбор делается заново на вехе `external-view` |
| [[docs/decisions/roman-inspired-polity-is-the-domain\|roman-inspired-polity-is-the-domain]] | Superseded | Римско-вдохновлённая полития является предметной областью |
| [[docs/decisions/project-memory-and-documentation-lifecycle\|project-memory-and-documentation-lifecycle]] | Superseded | Разделить оперативную память, документы и историю решений |
| [[docs/decisions/roman-names-in-presentation-only\|roman-names-in-presentation-only]] | Rejected | Отклонено: римские названия только в presentation layer |

## Рассмотрено и отклонено

Не решения, а память об отказах: чтобы следующая сессия не предлагала это
заново как свежую мысль. Развёрнутое обоснование — по ссылке.

| Отклонено | Почему | Где рассуждение |
| --- | --- | --- |
| Идеологические координаты, политический компас | измеряют взгляды, а у нас нет ни партий, ни выборов, ни общественного мнения; мы измеряем институты | [[docs/research/regime-measurement\|regime-measurement]] |
| Балл или индекс демократии игроку | превращает игру в оптимизацию числа; приз партии — диагноз | там же |
| Auctor решает дела с карточки, как игрок Reigns | ломает разделение полномочий на уровне игрока; карточка меняет закон, а не исход дела | [[docs/decisions/player-works-a-stack-of-proposals\|player-works-a-stack-of-proposals]] |
| Несколько толкований одной нормы | норма вводится структурой с подтверждением игрока, авторского текста как источника нет, толкованиям неоткуда взяться | [[docs/research/legalruleml\|legalruleml]] |
| Сам стандарт LegalRuleML | формат обмена размеченным настоящим законодательством; у нас одна юрисдикция и корпус, порождаемый кодом | там же |
| Полная деонтическая логика | рассуждений о нормах как таковых нет; нужны обязанности как объекты, а не модальные операторы | [[docs/decisions/duties-have-a-lifecycle\|duties-have-a-lifecycle]] |
| Стабильные модели | несколько прочтений корпуса против обещания одного ответа с трассой; и NP-трудно | [[docs/decisions/norms-are-inference-rules\|norms-are-inference-rules]] |
| Отказ принимать противоречивый корпус | замыкает правовую базу: законодатель обязан был бы писать непротиворечиво, а игра про обратное | там же |
| Well-founded поверх стратификации | два механизма там, где хватает одного вытеснения | там же |
| Проза как источник нормы, вычисление условия моделью в рантайме | ломает воспроизведение периода и делает текст модели нормой | там же |
| Эмбеддинги как представление нормы | в векторе не обнаружить противоречие, не объяснить вывод, не воспроизвести | там же |
| Счёт внутри правил вывода | за счётом придёт агрегация и собственный порядок вычисления; считает линтер | [[docs/design/components/deduction\|deduction]] |
| Парсер языка норм | стоп-правило компонента | там же |
| Извлечённое из журнала дерево решает дела | прецедент через чёрный ход: прошлые выходы модели стали бы нормой | [[docs/decisions/practice-becomes-norm-only-by-enactment\|practice-becomes-norm-only-by-enactment]] |
| Запрет переворота нормой в движке | римская конституция такой нормы не имела; защита была структурной, а вшитый запрет делает провал недостижимым | [[docs/design/casus/decemvirate\|decemvirate]] |
| «Вне корпуса не происходит ничего» | игра лишается сюжета о смерти конструкции | там же |
| «Победитель переписывает корпус» | реалистичнее всего и дороже всего | там же |
| Вещное и обязательственное право | граница проходит по статусу: вещи и сделки — не наше | [[docs/research/roman-private-law-cases\|roman-private-law-cases]] |
| Механизм специальности нормы | был бы третьим генератором пар вытеснения — «частное бьёт общее»; при надобности отдельное решение | [[docs/decisions/document-hierarchy-fills-the-defeat-relation\|document-hierarchy-fills-the-defeat-relation]] |
| Номера как идентификаторы решений и понятий | у одного автора и неизменяемых решений не дают ни защиты от коллизий, ни устойчивости идентичности, а в тексте ссылки читаются хуже слага | [[docs/decisions/documentation-layers\|documentation-layers]] |
| Вся документация по четырём режимам Diátaxis | режимы чтения не выражают связность компонентов и воспроизводят перекрёстные ссылки | [[docs/decisions/design-layer-has-a-spine-and-component-cards\|design-layer-has-a-spine-and-component-cards]] |

## Что здесь требует внимания автора

Решений в статусе `Proposed` нет. Четыре решения —
`player-works-a-stack-of-proposals`, `extension-points-require-a-casus`,
`document-hierarchy-fills-the-defeat-relation` и
`design-layer-has-a-spine-and-component-cards` — приняты: `Accepted` проставлен
помощником по прямому указанию автора, и это записано в поле `Accepted by`.
`template.md` носит статус `Proposed` по устройству, а не потому, что чего-то
ждёт.

Четыре решения **отложены и ещё не написаны** — их список в
[[docs/open-questions|открытых вопросах]].

**Атрибуция.** Часть принятых решений унаследована с пометкой `Deciders:
author`, но авторство не подтверждено — у них стоит `Proposed by: не
подтверждено`. Подтверждены как авторские: `durable-commit-is-a-semantic-contract`,
`application-owns-workflow-state`, `proto-first-external-api`. Решение
`llm-does-not-grant-authority` предложено помощником и принято автором явно.

**Долг проверки.** У `llm-does-not-grant-authority` критерий гласит «до принятия
построить сценарий, где более сильная модель пытается получить выгодный исход за
пределами компетенции». Сценарий не построен; решение принято. Нужно либо
построить его, либо переформулировать критерий как последующую проверку.
