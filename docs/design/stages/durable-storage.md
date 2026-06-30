---
tags:
  - устройство
  - веха
---

# durable-storage — хранение

Зависит от: [[docs/design/stages/playable-case|playable-case]]  
Компоненты: [[docs/design/components/storage|storage]]

## Состав

Реальный адаптер вместо памяти.

## Демонстрация

Та же партия на реальном адаптере.

## Гейт

Контрактные тесты durable commit против реального окружения, а не эмулятора —
см. [[docs/design/testing|testing]].
