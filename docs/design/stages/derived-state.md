---
tags:
  - устройство
  - веха
---

# derived-state — состояние выводится

Зависит от: [[docs/design/stages/derivation|derivation]], [[docs/design/stages/scenario-loader|scenario-loader]]  
Компоненты: [[docs/design/components/corpus|corpus]], [[docs/design/components/facts|facts]], [[docs/design/components/entitlement|entitlement]], [[docs/design/components/scenario|scenario]]

## Состав

Корпус с версиями и срезом на период, неизменяемые факты с провенансом,
`entitlement` как единственный ответчик, загрузка сценария.

## Демонстрация

Правила [[docs/design/casus/citizenship|казуса о гражданстве]] дают Марку статус,
удаление `A4` его снимает.

## Гейт

Сценарий казуса проходит на **подменённой стратегии разрешения норм**. Без него
первый шов фиктивен, а обязательство заменяемости остаётся словами.
