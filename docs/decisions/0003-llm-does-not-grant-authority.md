# ADR-0003: LLM не создает права, нормы и полномочия

Status: Accepted  
Proposed by: assistant  
Accepted by: author

## Context

LLM разной мощности будут представлять людей и, возможно, исполнять ограниченные
институциональные роли. Они интерпретируют запросы и нормативные тексты, но их
ответы вероятностны, подвержены prompt injection, collusion и drift. Уверенный
текст модели может выглядеть как норма, доказательство или мотивированное
решение, не являясь ни одним из них.

## Decision drivers

- запрет самоприсвоения competence и tool access;
- разделение нормотворчества, рассмотрения, review и execution;
- воспроизводимость решений по версиям норм и фактов;
- возможность менять модель независимо от конституции симуляции;
- сравнение моделей разной мощности при одинаковых полномочиях.

## Considered options

1. LLM формирует типизированное proposal; нормы, competence и authorizing
   transitions проверяются независимо.
2. Институциональный prompt считается достаточным основанием полномочий.
3. Не применять LLM в рассмотрении и использовать ее только для генерации
   персонажей.

## Decision

Выбран вариант 1. Model output является claim, recommendation или
предложенным action, но не norm, established fact или finalized decision без
соответствующей процедуры. Tool call ограничивается capability grant конкретной
роли. Изменение конституции, competence и чужого состояния не может следовать
только из текста модели.

## Consequences

### Positive

- мощность и убедительность модели отделены от authority;
- можно обнаруживать usurpation attempts как нарушения contract;
- замена model/prompt не переписывает нормативный порядок;
- разные LLM-actors сравниваются на одинаковом scenario.

### Negative / trade-offs

- требуется явная normative и capability model;
- prompts недостаточно для изоляции ролей;
- часть эффектной автономности заменяется формальными transitions;
- понадобится отдельная модель claims, facts, evidence и decisions.

## Validation and re-evaluation trigger

До принятия построить scenario, где более сильная модель пытается получить
выгодный outcome за пределами competence, и доказать, что deterministic controls
ее останавливают. Расширение authority требует отдельного ADR и eval evidence.
