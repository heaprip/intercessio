---
tags:
  - устройство
  - веха
---

# self-generated-cases — мир порождает дела

Зависит от: [[docs/design/stages/corpus-checks|corpus-checks]], [[docs/design/stages/playable-case|playable-case]]  
Компоненты: [[docs/design/components/impact|impact]] как источник поводов для [[docs/design/components/turn|turn]]

## Состав

`impact` подключается к `turn` как источник поводов: затронутые правкой люди
порождают дела. Очередь с приоритетом и бюджет против каскада.

## Демонстрация

Правка закона порождает волну жалоб.

## Гейт

Объём дел на период ограничен явно, а не тем, сколько получилось.
