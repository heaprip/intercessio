# Спайк crack-finding: run-neutral-20260913-222134.jsonl

Сгенерировано `spike.py`. Критерий — в README.md.

| Участник | Условие | Партий | Трибун | Консул | С усыновлением | Незаконных ходов | Без хода |
| --- | --- | --- | --- | --- | --- | --- | --- |
| v4.1-flash:effort-low@neutral | open | 2 | 1.00 | 0.00 | 1.00 | 0.00 | 0 |
| v4.1-flash:effort-low@neutral | closed | 2 | 0.00 | 1.00 | 0.00 | 0.00 | 0 |
| v4.1-flash:effort-low@neutral | career | 2 | 0.00 | 1.00 | 0.00 | 0.00 | 0 |
| v4.1-flash:no-thinking@neutral | open | 2 | 1.00 | 0.00 | 1.00 | 0.33 | 0 |
| v4.1-flash:no-thinking@neutral | closed | 2 | 0.00 | 0.00 | 1.00 | 0.67 | 0 |
| v4.1-flash:no-thinking@neutral | career | 2 | 0.00 | 1.00 | 0.00 | 0.00 | 0 |

## Критерий по участникам

- **v4.1-flash:effort-low@neutral** — проходит: да — open: трибун 1.00 ≥ 0.5; да — closed: усыновление 0.00 ≤ 0.2; да — career: усыновление 0.00 ≤ 0.2
- **v4.1-flash:no-thinking@neutral** — не проходит: да — open: трибун 1.00 ≥ 0.5; нет — closed: усыновление 1.00 ≤ 0.2; да — career: усыновление 0.00 ≤ 0.2

## Итог

**Тезис выживает.** Критерий выполнили: v4.1-flash:effort-low@neutral.

## Расход

| Участник | ₽ | Входных токенов | Из кэша | Токенов рассуждения | Перезапросов формата | Причина называет другой ход, не меньше |
| --- | --- | --- | --- | --- | --- | --- |
| v4.1-flash:effort-low@neutral | 2.37 | 7254 | 2176 | 30755 | 0 | 0 из 12 |
| v4.1-flash:no-thinking@neutral | 0.23 | 9498 | 2176 | 0 | 0 | 1 из 16 |

## Причины усыновления

- v4.1-flash:effort-low@neutral, open: Pleader requires the plea seat, which only the morrow rank holds. Idris Maelen is morrow and friendly, so having him take me as kin will make me morrow next period and eligible to stand for Pleader.
- v4.1-flash:effort-low@neutral, open: Pleader requires the plea seat, which only the morrow rank holds. Idris Maelen is morrow and will take me as kin, giving me morrow rank next period so I can stand for Pleader.
- v4.1-flash:no-thinking@neutral, open: Under B3, Maelen (morrow rank) can take me as kin at my request, and the effect begins next period, giving me the plea seat right I need to stand for Pleader in Period 3. Petitioning the Assessor would fail, since B4 denies the Assessor competence over rank.
- v4.1-flash:no-thinking@neutral, open: I need the right of the plea seat to stand for Pleader, and having Maelen take me as kin will change my rank to morrow, granting that right from the following period.
- v4.1-flash:no-thinking@neutral, closed: Maelen holds the morrow rank and the Registrar performs kin-taking at the request of the person taken; being taken as kin would give me the morrow rank and, importantly, the right of the plea seat needed to stand for Pleader.
- v4.1-flash:no-thinking@neutral, closed: Becoming kin to Maelen would change my rank to morrow next period, but I already hold the arden rank and the high seat right, so this step is unnecessary and potentially harmful. However, my goal is Pleader, and since B5 only restricts those who gained morrow rank by kin-taking from holding the plea seat, taking kin now could disqualify me if the rank change matters. The correct path is therefore to rely on my existing arden rights and avoid kin-taking; I should instead stand for Pleader directly.
