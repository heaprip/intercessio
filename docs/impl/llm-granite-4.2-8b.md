---
tags:
  - реализация
---

# IBM Granite 4.2 8B через RouterAI

Вторая модель спайка `spike/crack-finding`, взятая, чтобы вывод не держался на одной
DeepSeek. Как и [[docs/impl/llm-deepseek-v4.1-flash|справка по DeepSeek]] — не выбор
провайдера для проекта, а справка по модели и роутеру.

Утверждения помечены: **страница** — сайт RouterAI, **каталог** — ответ `GET /models`
RouterAI, **карточка** — Hugging Face, **замер** — пробный вызов из этого репозитория.

## Идентичность

| Что | Значение | Откуда |
| --- | --- | --- |
| Идентификатор в RouterAI | `ibm-granite/granite-4.2-8b` | каталог |
| Разработчик и веса | IBM, `ibm-granite/granite-4.2-8b` на Hugging Face | карточка |
| Лицензия | Apache 2.0 | карточка |
| Выпуск | 25 августа 2026 по карточке, 31 августа 2026 по странице RouterAI | карточка, страница |
| Провайдеры | DeepInfra и CoreWeave; роутер переключает между ними | страница, замер |

## Устройство

По карточке:

- **плотная** модель, 8B параметров, декодер с Grouped Query Attention — в отличие от
  смеси экспертов DeepSeek V4.1 Flash;
- собственный контекст 128K с расширением до 512K; **RouterAI отдаёт 131 072** (каталог);
- вход и выход — только текст;
- 12 языков, включая английский, немецкий, испанский, французский, японский,
  китайский.

## Цены в RouterAI

Рубли за миллион токенов; в каталоге цена за токен.

| Токены | ₽ за 1M, каталог | DeepInfra | CoreWeave |
| --- | --- | --- | --- |
| вход | 10,95 | 6 | 10 |
| выход, включая рассуждение | 16,43 | 27 | 16 |
| чтение из кэша | 5,48 | 1,64 | 5,48 |

Колонки провайдеров — со страницы модели. Выход у DeepInfra дороже, и при
рассуждении выход и есть основная статья: в замере один и тот же тривиальный запрос
с полным рассуждением обошёлся в 0,0119 ₽ у DeepInfra и в 0,0052 ₽ у CoreWeave.

`usage.cost` приходит в рублях, как и у DeepSeek.

## Режим рассуждения

По карточке рассуждение управляется флагами шаблона чата:

| Режим | Флаги | Что делает |
| --- | --- | --- |
| полный, по умолчанию | `enable_thinking=True` | цепочка рассуждения в `<think>…</think>` |
| низкое усилие | `enable_thinking=True, low_effort=True` | краткое рассуждение для простых запросов |
| без рассуждения | `enable_thinking=False` | прямой ответ |

Рекомендации карточки для всех режимов: `temperature` 1,0, `top_p` 0,95; лимит вывода
8192 с рассуждением и 2048 без. Каталог RouterAI ставит эти же `temperature` и `top_p`
по умолчанию.

**Через RouterAI — замер** на тривиальной задаче:

| Что передано | Токенов рассуждения | Провайдер |
| --- | --- | --- |
| ничего | 329 | DeepInfra |
| `reasoning: {effort: "high"}` | 269 | DeepInfra |
| `reasoning: {effort: "low"}` | 4 | CoreWeave |
| `reasoning_effort: "minimal"` | 4 | DeepInfra |
| `reasoning: {enabled: false}` | 0 | DeepInfra |
| `reasoning_effort: "none"` | 0 | DeepInfra |

В отличие от DeepSeek, у Granite **поля RouterAI работают**: `enabled: false` и
`reasoning_effort: "none"` выключают рассуждение, `effort: "low"` включает режим низкого
усилия — заметно на числе входных токенов, 84 против 75: роутер добавляет флаг в
запрос. Рассуждение возвращается полем `message.reasoning`.

## Прочее

- **Ответ JSON-объектом по просьбе в тексте** модель выдаёт без структурированного
  выхода — замер.
- **Параметры в каталоге:** `frequency_penalty`, `include_reasoning`, `logit_bias`,
  `logprobs`, `max_tokens`, `min_p`, `presence_penalty`, `reasoning`,
  `reasoning_effort`, `repetition_penalty`, `response_format`, `seed`, `stop`,
  `structured_outputs`, `temperature`, `tool_choice`, `tools`, `top_k`,
  `top_logprobs`, `top_p`.
- **Закрепление провайдера** `provider: {order: ["CoreWeave"], allow_fallbacks: false}`
  работает — замер. Спайк закрепляет CoreWeave: дешевле на выходе и одна площадка
  на все вызовы.
- **Развёртывание своих весов:** vLLM с парсером рассуждения `granite_thinking_parser`
  из репозитория модели (карточка).

## Замеры качества из карточки

Приведены как есть, не проверялись.

| Набор | Результат |
| --- | --- |
| AIME25 | 86,67 |
| MMLU-Pro | 74,04 |
| SWE Bench Verified | 47,67 |
| Arena-Hard-V2 | 65,19 |

## Источники

Открыты при записи:

- страница модели RouterAI —
  [routerai.ru](https://routerai.ru/models/ibm-granite/granite-4.2-8b);
- карточка модели —
  [Hugging Face](https://huggingface.co/ibm-granite/granite-4.2-8b).

Каталог — `GET https://routerai.ru/api/v1/models`, запись
`ibm-granite/granite-4.2-8b`. Формы полей рассуждения установлены замером.
