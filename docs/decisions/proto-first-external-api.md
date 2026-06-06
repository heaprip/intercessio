---
tags:
  - adr
  - решение
---

# Внешний API строится proto-first, REST через gRPC-Gateway

Status: Accepted  
Proposed by: author  
Accepted by: author  
Зависит от: —  
Заменяет: —

> **Требует пересмотра.** Решение принималось под прежнюю рамку — внешних
> интеграторов. У игры один потребитель, собственный фронт, и поток отчётов
> периода вместо ресурсно-ориентированного REST. Сохранено как есть до
> появления первого реального контракта на вехе M7; см.
> [[docs/open-questions|открытые вопросы]].

## Context

Система должна предоставлять обычный HTTP/JSON API и сохранять возможность
native gRPC. Envoy transcoding плохо подходит как обязательная часть
serverless-развёртывания. ConnectRPC упрощает единый RPC-over-HTTP handler, но
не даёт ресурсно-ориентированный REST-контракт и OpenAPI без дополнительных
инструментов.

## Decision

Внешний синхронный контракт описывается `.proto`. Buf управляет lint, breaking
checks, зависимостями и закреплённой по версиям генерацией. Публичные HTTP
bindings задаются явно через `google.api.http`. Из схемы генерируются Go-типы,
gRPC stubs, gRPC-Gateway handlers и OpenAPI artifact.

REST и native gRPC являются driving-адаптерами над одним use case. Protobuf
messages не становятся доменными типами. Модули монолита вызывают application
API напрямую, а не через loopback HTTP/gRPC.

Direct in-process регистрация gRPC-Gateway не выполняет gRPC-интерсепторы,
поэтому авторизация, идемпотентность и критическая валидация принадлежат
application policy, а не только transport middleware.

## Consequences

Одна схема формирует оба адаптера, Envoy не требуется. Цена: одновременно
поддерживаются protobuf, JSON/HTTP и OpenAPI; metadata, trailers, streaming и
error details переводятся не полностью; часть HTTP-семантики проектируется
вручную.

## Validation

Первый публичный API проходит HTTP- и gRPC-контрактные тесты над одним use
case. Пересмотреть, если API окажется полностью RPC-first или streaming станет
доминирующим.

## Links

- Детали транспорта и генерации: [[docs/impl/api-transport|impl]]
