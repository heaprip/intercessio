---
tags:
  - adr
  - решение
---

# Отделить доменные события от транспорта

Status: Accepted  
Proposed by: не подтверждено — см. [[docs/open-questions|открытые вопросы]]  
Accepted by: author  
Зависит от: —  
Заменяет: —

## Context

Архитектура должна выражать event-oriented поведение без зависимости ядра от
broker, serialization format или runtime. Требуются сценарные тесты in-process.

## Decision drivers

- чистая доменная модель;
- deterministic simulation;
- возможность менять transport;
- отдельная эволюция внешних контрактов и внутренних фактов.

## Considered options

1. Доменные события как Go values, mapping на transport message в adapter.
2. Использовать broker envelope непосредственно в домене.
3. Не моделировать факты, вызывать все реакции напрямую.

## Decision

Доменное событие — типизированный факт в прошедшем времени, созданный доменной
моделью. Оно не содержит broker metadata и не сериализует себя. Application
layer решает, какие реакции выполнить, а outbound adapter преобразует только
необходимые факты во внешние сообщения.

Это решение не требует event sourcing и не означает, что все межмодульные
взаимодействия асинхронны.

## Consequences

### Positive

- ядро и тесты не требуют инфраструктуры;
- внешняя схема не протекает во внутреннюю модель;
- можно редактировать PII и версионировать интеграционный контракт отдельно.

### Negative / trade-offs

- появится явный mapping;
- нужно определить transaction boundary и момент publication;
- семантика retry/idempotency остается обязанностью adapter/use case.

## Validation and re-evaluation trigger

Проверить на первом вертикальном срезе: один и тот же сценарий должен работать с
in-memory collector без transport. Пересмотреть, если mapping оказывается чистым
дублированием без независимой эволюции на нескольких реальных интеграциях.
